package server

import (
	"embed"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/ChancellorCookie/fulla/internal/api"
	"github.com/ChancellorCookie/fulla/internal/core"
	"github.com/ChancellorCookie/fulla/internal/db"
)

//go:embed all:frontend/build/*
var frontend embed.FS

// Run starts the HTTP server
func Run() error {
	dataDir := env("FULLA_DATA_DIR", "./data")
	dbPath := env("FULLA_DB_PATH", dataDir+"/fulla.db")
	host := env("FULLA_HOST", "0.0.0.0")
	port := env("FULLA_PORT", "4212")

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	// Open DB
	database, err := db.New(dbPath)
	if err != nil {
		return err
	}
	defer database.Close()

	// Process due recurring transactions
	n, err := database.ProcessDueRecurrings()
	if err != nil {
		log.Printf("WARN: process recurrings: %v", err)
	} else if n > 0 {
		log.Printf("Processed %d recurring transactions", n)
	}

	// Seed default categories if none exist
	seedDefaultData(database)

	// Setup routes
	h := &api.Handler{DB: database}
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", h.Health)
	mux.HandleFunc("/api/accounts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			h.ListAccounts(w, r)
		case "POST":
			h.CreateAccount(w, r)
		default:
			w.WriteHeader(405)
		}
	})
	mux.HandleFunc("/api/accounts/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			h.DeleteAccount(w, r)
		} else {
			w.WriteHeader(405)
		}
	})
	mux.HandleFunc("/api/categories", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			h.ListCategories(w, r)
		case "POST":
			h.CreateCategory(w, r)
		default:
			w.WriteHeader(405)
		}
	})
	mux.HandleFunc("/api/categories/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "DELETE":
			h.DeleteCategory(w, r)
		case "PATCH":
			h.UpdateCategory(w, r)
		default:
			w.WriteHeader(405)
		}
	})
	mux.HandleFunc("/api/transactions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			h.ListTransactions(w, r)
		case "POST":
			h.CreateTransaction(w, r)
		default:
			w.WriteHeader(405)
		}
	})
	mux.HandleFunc("/api/transactions/bulk", h.BulkUpdateTransactions)
	mux.HandleFunc("/api/transactions/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/transactions/")
		if strings.HasPrefix(path, "toggle-exclude/") {
			id := strings.TrimPrefix(path, "toggle-exclude/")
			id = strings.TrimSuffix(id, "/")
			if err := h.DB.ToggleExcludeTransaction(id); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"toggled"}`))
			return
		}
		if r.Method == "DELETE" {
			h.DeleteTransaction(w, r)
		} else if r.Method == "PATCH" {
			h.UpdateTransaction(w, r)
		} else {
			w.WriteHeader(405)
		}
	})
	mux.HandleFunc("/api/summary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			h.MonthSummary(w, r)
		} else {
			w.WriteHeader(405)
		}
	})
	mux.HandleFunc("/api/forecast", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			h.Forecast(w, r)
		} else {
			w.WriteHeader(405)
		}
	})
	mux.HandleFunc("/api/recurring", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			h.ListRecurrings(w, r)
		case "POST":
			h.CreateRecurring(w, r)
		default:
			w.WriteHeader(405)
		}
	})
	mux.HandleFunc("/api/recurring/", func(w http.ResponseWriter, r *http.Request) {
		// Check for /api/recurring/{id}/history
		path := strings.TrimPrefix(r.URL.Path, "/api/recurring/")
		if strings.HasSuffix(path, "/history") {
			if r.Method == "GET" {
				h.GetRecurringHistory(w, r)
				return
			}
			w.WriteHeader(405)
			return
		}
		switch r.Method {
		case "DELETE":
			h.DeleteRecurring(w, r)
		case "POST":
			h.ToggleRecurring(w, r)
		case "PATCH":
			h.UpdateRecurring(w, r)
		default:
			w.WriteHeader(405)
		}
	})
	mux.HandleFunc("/api/recurring/process", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			h.ProcessRecurrings(w, r)
		} else {
			w.WriteHeader(405)
		}
	})

	// CSV Import
	mux.HandleFunc("/api/import/csv", h.ImportCSV)

	// Annual
	mux.HandleFunc("/api/annual", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			h.AnnualSummary(w, r)
		} else {
			w.WriteHeader(405)
		}
	})

	// Static frontend (SvelteKit build output) with SPA fallback
	staticFS, err := fs.Sub(frontend, "frontend/build")
	if err != nil {
		log.Printf("WARN: no frontend build embedded: %v (API-only mode)", err)
	} else {
		fileServer := http.FileServer(http.FS(staticFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// API routes are handled above; anything reaching here is static
			// Serve directly — FileServer handles existing files
			// For SPA client routes (paths without file extension), serve index.html
			path := r.URL.Path
			if path != "/" && !strings.Contains(path, ".") {
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	addr := net.JoinHostPort(host, port)
	log.Printf("Fulla starting on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// seedDefaultData creates initial categories + default account if tables are empty
func seedDefaultData(database *db.DB) {
	cats, _ := database.GetCategories(nil)
	accts, _ := database.GetAccounts()
	rts, _ := database.GetRecurrings()

	if len(cats) == 0 {
		seedCategories(database)
	}
	if len(accts) == 0 {
		database.CreateAccount(core.CreateAccountRequest{
			Name: "Sparkasse Giro", Type: "checking", Currency: "EUR",
		})
		log.Printf("Seeded default account: Sparkasse Giro")
	}
	if len(rts) == 0 {
		seedDemoRecurrings(database)
	}

	txs, _ := database.GetTransactions("", "", "", "", 1)
	if len(txs) == 0 {
		seedDemoTransactions(database)
	}
}

func seedDemoRecurrings(database *db.DB) {
	accts, _ := database.GetAccounts()
	cats, _ := database.GetCategories(nil)
	if len(accts) == 0 || len(cats) == 0 { return }

	aid := accts[0].ID
	catMap := map[string]string{}
	for _, c := range cats { catMap[c.Name] = c.ID }

	demos := []struct{ name, cat string; amt float64; interval string }{
		// Income
		{"Lohn", "Einkommen", 3338, "monthly"},
		// Versicherungen
		{"KH-Tagegeld (G)", "Versicherungen", -22, "monthly"},
		{"AU/Pflege/Sterbe", "Versicherungen", -140, "monthly"},
		{"Kasko VW", "Versicherungen", -78, "monthly"},
		{"HP/HR/Reise", "Versicherungen", -35, "monthly"},
		{"Unfall", "Versicherungen", -35, "monthly"},
		{"Fahrrad Enra", "Versicherungen", -12.50, "monthly"},
		{"Verrechnung Mama", "Versicherungen", 60, "monthly"},
		// Kfz/Mobilität
		{"Sprit", "Kfz/Mobilität", -70, "monthly"},
		{"Steuer VW", "Kfz/Mobilität", -4.17, "monthly"},
		{"ADAC", "Kfz/Mobilität", -7.83, "monthly"},
		{"Fahrrad Lea", "Kfz/Mobilität", -200, "monthly"},
		// Wohnung
		{"Miete Warm", "Wohnung", -530, "monthly"},
		{"Strom", "Wohnung", -82, "monthly"},
		{"GEZ", "Wohnung", -18.33, "monthly"},
		{"Haushaltsgeld", "Wohnung", -50, "monthly"},
		// Internet/Handy
		{"Hoher Weg", "Internet/Handy", -55, "monthly"},
		{"Max Handy", "Internet/Handy", -35, "monthly"},
		{"Mama Handy", "Internet/Handy", -5, "monthly"},
		{"Prinzen", "Internet/Handy", -54, "monthly"},
		// Vermögen
		{"Riester", "Vermögen", -160, "monthly"},
		{"ETF", "Vermögen", -300, "monthly"},
		{"Sparen/HK", "Vermögen", -350, "monthly"},
		// Unterhaltung
		{"Patreon", "Unterhaltung", -2, "monthly"},
		{"Proton", "Unterhaltung", -5, "monthly"},
		{"Strato", "Unterhaltung", -0.83, "monthly"},
		{"YT", "Unterhaltung", -13, "monthly"},
	}
	for _, d := range demos {
		cid := catMap[d.cat]
		if cid == "" { continue }
		database.CreateRecurring(core.CreateRecurringRequest{
			AccountID: aid, CategoryID: cid,
			Amount: d.amt, Description: d.name,
			IntervalKind: d.interval,
		})
	}
	log.Printf("Seeded %d demo recurring transactions", len(demos))
}

func seedCategories(database *db.DB) {

	builtin := []struct {
		Name     string
		Color    string
		Icon     string
		IsIncome bool
	}{
		// Income
		{"Einkommen", "#4a7c59", "💰", true},
		// Expenses
		{"Versicherungen", "#6b8f71", "🛡️", false},
		{"Kfz/Mobilität", "#5c7a99", "🚗", false},
		{"Wohnung", "#8c6d8c", "🏠", false},
		{"Internet/Handy", "#8c8c4a", "📱", false},
		{"Vermögen", "#c4854a", "📈", false},
		{"Unterhaltung", "#8c5c5c", "🎮", false},
		{"Lebensmittel", "#6b6b4a", "🛒", false},
		{"Gesundheit", "#4a7c6b", "💊", false},
		{"Sonstige Ausgaben", "#5c6b6b", "📦", false},
	}
	for _, c := range builtin {
		database.CreateCategory(core.CreateCategoryRequest{
			Name:     c.Name,
			Color:    c.Color,
			Icon:      c.Icon,
			IsIncome: c.IsIncome,
		})
	}
	log.Printf("Seeded %d default categories", len(builtin))
}

func seedDemoTransactions(database *db.DB) {
	accts, _ := database.GetAccounts()
	cats, _ := database.GetCategories(nil)
	if len(accts) == 0 || len(cats) == 0 { return }

	aid := accts[0].ID
	catMap := map[string]string{}
	for _, c := range cats { catMap[c.Name] = c.ID }

	type txSeed struct {
		date, desc, cat string
		amt             float64
		recurMatchID    string // "manual" to exclude from recurring matching
	}
	august := []txSeed{
		// 2026-08-03
		{"2026-08-03", "4.9085369.43 08/2026 160,42 4.9438661.73 08/2026 142,56 4.9641716.73 08/2026 8,57 — Generali Deutschland Lebensversicherung AG", "Vermögen", -311.55, "manual"},
		{"2026-08-03", "Generali Deutschland Krankenversicherung AG Versicherungsnr.81522464 Beitrag Krankenversicherung — Generali Deutschland Krankenversicherung AG", "Gesundheit", -26.25, "manual"},
		{"2026-08-03", "ADAC E.V. Jochums Max BEITRAG: 01.08.26-31.07.27 — Allg.Deutscher Automobil-Club ADAC e.V.", "Kfz/Mobilität", -129.00, "manual"},
		{"2026-08-03", "010115314040 HAUSR/GLAS/HAFT/UNF/REISE 010826 — Generali Deutschland Versicherung AG", "Versicherungen", -75.29, "manual"},
		{"2026-08-03", "Entgeltabrechnung siehe Anlage", "Sonstige Ausgaben", -5.90, ""},
		{"2026-08-03", "2026-07-31T14:20 Debitk.3 2026-12 — ALDI SE U. CO. KG/KRUPPSTR. 51/DUISBURG/DE", "Lebensmittel", -9.21, ""},
		{"2026-08-03", "2026-08-01T10:22 Debitk.3 2026-12 — Nanu-Nana//Moers/DE", "Sonstige Ausgaben", -9.85, ""},
		{"2026-08-03", "2026-07-31T14:36 Debitk.3 2026-12 — Gartencenter Schloesser//Moers/DE", "Wohnung", -50.98, ""},
		// 2026-08-04
		{"2026-08-04", "LASTSCHRIFT easybank — easybank", "Sonstige Ausgaben", -15.00, ""},
		// 2026-08-05
		{"2026-08-05", "X70278 531 245 0X DATUM 05.08.2026, 15.34 UHR — Zentrale Zahlstelle Justiz", "Sonstige Ausgaben", -482.00, ""},
		{"2026-08-05", "IWishThisWasForNFLTickets DATUM 05.08.2026, 11.11 UHR — Marcus Goeddecke", "Wohnung", -100.00, ""},
		{"2026-08-05", "302-5640475-8638740 AMZN Mktp DE JT9DXSH19X2ZO44R — AMAZON PAYMENTS EUROPE S.C.A.", "Sonstige Ausgaben", -26.30, ""},
		{"2026-08-05", "1052116795310/PP.6292.PP/. , Ihr Einkauf bei — PayPal Europe S.a.r.l. et Cie S.C.A", "Sonstige Ausgaben", -25.00, ""},
		// 2026-08-06
		{"2026-08-06", "06.08/17.36UHR OSTRING RC — GA NR00002205 BLZ35450000 3", "Unterhaltung", -150.00, ""},
		{"2026-08-06", "360/053784502 VermogensSparplan — Max Jochums", "Vermögen", -99.99, "manual"},
		{"2026-08-06", "360/053784503 VermogensSparplan — Max Jochums", "Vermögen", -99.98, "manual"},
		{"2026-08-06", "360/053784501 VermogensSparplan — Max Jochums", "Vermögen", -99.98, "manual"},
		// 2026-08-07
		{"2026-08-07", "2026-08-06T13:34 Debitk.3 2026-12 — FRISCHECENTER GERDES//MOERS/DE", "Lebensmittel", -72.42, ""},
		{"2026-08-07", "2026-08-06T12:23 Debitk.3 2026-12 — ALDI SE U. CO. KG/DRENNESWEG 3/MOERS/DE", "Lebensmittel", -33.18, ""},
		// 2026-08-10
		{"2026-08-10", "925511 DATUM 10.08.2026, 09.18 UHR — Kartbaan Winterswijk", "Unterhaltung", -820.20, ""},
		{"2026-08-10", "INSTANT TRANSFER — PAYPAL", "Einkommen", 600.45, ""},
		{"2026-08-10", "2026-08-07T14:35 Debitk.3 2026-12 — ALDI SAGT DANKE 01 066//Borken/DE", "Lebensmittel", -60.09, ""},
		{"2026-08-10", "Trip — MAX JOCHUMS", "Wohnung", 2000.00, "manual"},
		// 2026-08-11
		{"2026-08-11", "DRK-BEITRAG 1315009366 STRNR 131/5995/4407 FA: FINANZAMT WESEL — Deutsches Rotes Kreuz Kreisverband Niederrhein e.V.", "Sonstige Ausgaben", -10.00, "manual"},
	}

	count := 0
	for _, tx := range august {
		cid := catMap[tx.cat]
		if cid == "" {
			cid = catMap["Sonstige Ausgaben"]
		}
		database.CreateTransactionWithStatus(core.CreateTransactionRequest{
			AccountID:        aid,
			CategoryID:       cid,
			Amount:           tx.amt,
			Description:      tx.desc,
			Date:             tx.date,
			RecurringMatchID: tx.recurMatchID,
		}, "posted")
		count++
	}
	log.Printf("Seeded %d demo transactions", count)
}
