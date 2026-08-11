package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ChancellorCookie/fulla/internal/core"
	"github.com/ChancellorCookie/fulla/internal/db"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	DB *db.DB
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// resourceID extracts the last path segment as ID
func resourceID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// --- Health ---

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, core.HealthResponse{Status: "ok", Version: "0.1.0", Now: db.Now()})
}

// --- Accounts ---

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.DB.GetAccounts()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, accounts)
}

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req core.CreateAccountRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if req.Name == "" {
		writeError(w, 400, "name required")
		return
	}
	a, err := h.DB.CreateAccount(req)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, a)
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	id := resourceID(r.URL.Path)
	if err := h.DB.DeleteAccount(id); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// --- Categories ---

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	var income *bool
	if v := r.URL.Query().Get("income"); v == "true" {
		t := true
		income = &t
	} else if v == "false" {
		f := false
		income = &f
	}
	cats, err := h.DB.GetCategories(income)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, cats)
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req core.CreateCategoryRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if req.Name == "" {
		writeError(w, 400, "name required")
		return
	}
	c, err := h.DB.CreateCategory(req)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, c)
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := resourceID(r.URL.Path)
	if err := h.DB.DeleteCategory(id); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := resourceID(r.URL.Path)
	var req struct{ Name, Color string }
	if err := readJSON(r, &req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if err := h.DB.UpdateCategory(id, req.Name, req.Color); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "updated"})
}

// --- Transactions ---

func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("account")
	month := r.URL.Query().Get("month")
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	txs, err := h.DB.GetTransactions(accountID, month, status, search, 500)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, txs)
}

func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req core.CreateTransactionRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if req.AccountID == "" || req.CategoryID == "" || req.Date == "" {
		writeError(w, 400, "account_id, category_id, and date required")
		return
	}
	if req.Amount == 0 {
		writeError(w, 400, "amount required")
		return
	}
	tx, err := h.DB.CreateTransaction(req)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, tx)
}

func (h *Handler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	id := resourceID(r.URL.Path)
	if err := h.DB.DeleteTransaction(id); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (h *Handler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	id := resourceID(r.URL.Path)
	var req core.CreateTransactionRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if err := h.DB.UpdateTransaction(id, req); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "updated"})
}

// BulkUpdateTransactions handles PATCH /api/transactions/bulk
func (h *Handler) BulkUpdateTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PATCH" {
		w.WriteHeader(405)
		return
	}
	var req core.BulkUpdateRequest
	if err := readJSON(r, &req); err != nil {
		log.Printf("DEBUG bulk: invalid JSON: %v", err)
		writeError(w, 400, "invalid JSON")
		return
	}
	log.Printf("DEBUG bulk: ids=%d cat=%q status=%q", len(req.IDs), req.CategoryID, req.Status)
	n, err := h.DB.BulkUpdateTransactions(req)
	if err != nil {
		log.Printf("DEBUG bulk: error: %v", err)
		writeError(w, 500, err.Error())
		return
	}
	log.Printf("DEBUG bulk: updated %d rows", n)
	writeJSON(w, 200, map[string]int{"updated": n})
}

// --- Month Summary ---

func (h *Handler) MonthSummary(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("account")
	month := r.URL.Query().Get("month")
	if month == "" {
		writeError(w, 400, "month required (YYYY-MM)")
		return
	}
	s, err := h.DB.GetMonthSummary(accountID, month)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, s)
}

func (h *Handler) Forecast(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}
	s, err := h.DB.GetForecast(month)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, s)
}

// --- Recurring ---

func (h *Handler) ListRecurrings(w http.ResponseWriter, r *http.Request) {
	rts, err := h.DB.GetRecurrings()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, rts)
}

func (h *Handler) CreateRecurring(w http.ResponseWriter, r *http.Request) {
	var req core.CreateRecurringRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if req.AccountID == "" || req.CategoryID == "" || req.IntervalKind == "" {
		writeError(w, 400, "account_id, category_id, and interval_kind required")
		return
	}
	if req.Amount == 0 {
		writeError(w, 400, "amount required")
		return
	}
	rt, err := h.DB.CreateRecurring(req)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	// Retroactive matching: auto-post existing pending transactions that match
	matched, _ := h.DB.MatchPendingToRecurring(&rt)
	log.Printf("DEBUG recurring: created %s, retro-matched %d pending tx", rt.ID[:12], matched)

	writeJSON(w, 201, map[string]any{
		"recurring":      rt,
		"retro_matched": matched,
	})
}

func (h *Handler) ToggleRecurring(w http.ResponseWriter, r *http.Request) {
	id := resourceID(r.URL.Path)
	if err := h.DB.ToggleRecurring(id); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "toggled"})
}

func (h *Handler) DeleteRecurring(w http.ResponseWriter, r *http.Request) {
	id := resourceID(r.URL.Path)
	if err := h.DB.DeleteRecurring(id); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (h *Handler) UpdateRecurring(w http.ResponseWriter, r *http.Request) {
	id := resourceID(r.URL.Path)
	var req core.CreateRecurringRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if err := h.DB.UpdateRecurring(id, req); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "updated"})
}

func (h *Handler) ProcessRecurrings(w http.ResponseWriter, r *http.Request) {
	n, err := h.DB.ProcessDueRecurrings()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]int{"processed": n})
}

func (h *Handler) GetRecurringHistory(w http.ResponseWriter, r *http.Request) {
	id := resourceID(r.URL.Path)
	history, err := h.DB.GetRecurringHistory(id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, history)
}

// --- Annual ---

func (h *Handler) AnnualSummary(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		yearStr = time.Now().UTC().Format("2006")
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		writeError(w, 400, "invalid year")
		return
	}
	s, err := h.DB.GetAnnualSummary(year)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, s)
}
