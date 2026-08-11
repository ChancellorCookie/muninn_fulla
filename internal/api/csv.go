package api

import (
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ChancellorCookie/fulla/internal/core"
)

// ImportCSV handles CSV uploads. Auto-detects format:
//   - Sparkasse: semicolons, quoted fields, German dates/amounts
//   - Generic: commas, standard date/amount formats
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}

	accountID := r.URL.Query().Get("account")
	if accountID == "" {
		writeError(w, 400, "account query parameter required")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, 400, "failed to parse form: "+err.Error())
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "file field required")
		return
	}
	defer file.Close()

	// Detect delimiter before creating CSV reader
	comma := detectDelimiter(file)

	reader := csv.NewReader(file)
	reader.Comma = comma
	reader.LazyQuotes = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		writeError(w, 400, "failed to read CSV header: "+err.Error())
		return
	}

	// Map columns
	cols := mapColumns(header)

	imported := 0
	skipped := 0
	autoPosted := 0

	// Find default category for imported transactions
	defaultCat := h.findDefaultCategory(false)
	defaultCatIncome := h.findDefaultCategory(true)

	for {
		row, err := reader.Read()
		if err == io.EOF { break }
		if err != nil { skipped++; continue }

		tx, ok := parseRow(row, cols)
		if !ok { skipped++; continue }

		tx.AccountID = accountID
		tx.CategoryID = ""

		// Check for matching recurring transaction
		status := "pending"
		match, _ := h.DB.FindMatchingRecurring(tx.Description, tx.Amount)
		if match != nil {
			tx.CategoryID = match.CategoryID
			tx.RecurringMatchID = match.ID
			status = "posted"
		}

		// Assign default category if no match
		if tx.CategoryID == "" {
			if tx.Amount > 0 {
				tx.CategoryID = defaultCatIncome
			} else {
				tx.CategoryID = defaultCat
			}
		}

		_, err = h.DB.CreateTransactionWithStatus(tx, status)
		if err != nil { skipped++; continue }
		imported++
		if status == "posted" { autoPosted++ }
	}

	writeJSON(w, 200, map[string]int{
		"imported":    imported,
		"skipped":     skipped,
		"auto_posted": autoPosted,
	})
}

type csvCols struct {
	dateIdx       int
	descIdx       int
	amountIdx     int
	categoryIdx   int
	noteIdx       int
	payeeIdx      int // Sparkasse: Beguenstigter/Zahlungspflichtiger
	useTextIdx    int // Sparkasse: Verwendungszweck
	isGerman      bool
}

func mapColumns(header []string) csvCols {
	cols := csvCols{
		dateIdx: -1, descIdx: -1, amountIdx: -1,
		categoryIdx: -1, noteIdx: -1, payeeIdx: -1, useTextIdx: -1,
	}

	for i, h := range header {
		// Strip quotes from header
		h = strings.Trim(strings.ToLower(strings.TrimSpace(h)), `"`)
		switch {
		case h == "date" || h == "datum" || h == "buchungstag" || h == "valutadatum":
			cols.dateIdx = i
			if h == "buchungstag" || h == "valutadatum" || h == "datum" {
				cols.isGerman = true
			}
		case h == "description" || h == "beschreibung" || h == "buchungstext":
			cols.descIdx = i
		case h == "amount" || h == "betrag":
			cols.amountIdx = i
		case h == "category" || h == "kategorie":
			cols.categoryIdx = i
		case h == "note" || h == "notiz" || h == "info":
			cols.noteIdx = i
		case h == "verwendungszweck":
			cols.useTextIdx = i
		case h == "beguenstigter/zahlungspflichtiger" || h == "begünstigter/zahlungspflichtiger" ||
			h == "beguenstigter" || h == "zahlungspflichtiger" || h == "empfänger/auftraggeber":
			cols.payeeIdx = i
		}
	}

	if cols.amountIdx == -1 {
		cols.amountIdx = -1
	}
	return cols
}

func parseRow(row []string, cols csvCols) (core.CreateTransactionRequest, bool) {
	req := core.CreateTransactionRequest{}

	// Date
	if cols.dateIdx >= 0 && cols.dateIdx < len(row) {
		req.Date = parseDate(row[cols.dateIdx])
		if req.Date == "" {
			return req, false
		}
	} else {
		return req, false
	}

	// Amount
	if cols.amountIdx >= 0 && cols.amountIdx < len(row) {
		amt, err := parseAmount(row[cols.amountIdx], cols.isGerman)
		if err != nil || amt == 0 {
			return req, false
		}
		req.Amount = amt
	} else {
		return req, false
	}

	// Description: combine Verwendungszweck + Buchungstext + Payee
	var parts []string
	if cols.useTextIdx >= 0 && cols.useTextIdx < len(row) {
		if t := cleanSparkasseField(row[cols.useTextIdx]); t != "" {
			parts = append(parts, t)
		}
	}
	if cols.descIdx >= 0 && cols.descIdx < len(row) {
		if t := cleanField(row[cols.descIdx]); t != "" {
			parts = append(parts, t)
		}
	}
	if cols.payeeIdx >= 0 && cols.payeeIdx < len(row) {
		if t := cleanField(row[cols.payeeIdx]); t != "" {
			parts = append(parts, t)
		}
	}
	req.Description = strings.Join(parts, " — ")
	if req.Description == "" {
		req.Description = "Unbekannte Buchung"
	}

	// Note
	if cols.noteIdx >= 0 && cols.noteIdx < len(row) {
		req.Note = cleanField(row[cols.noteIdx])
	}

	return req, true
}

func cleanField(s string) string {
	return strings.Trim(strings.TrimSpace(s), `"`)
}

// cleanSparkasseField removes SVWZ+ and other SWIFT prefixes
func cleanSparkasseField(s string) string {
	s = cleanField(s)
	// Remove common SWIFT/MD flavors in Verwendungszweck
	for _, prefix := range []string{"SVWZ+", "EREF+", "KREF+", "MREF+", "CRED+", "ABWA+"} {
		s = strings.ReplaceAll(s, prefix, "")
	}
	return strings.TrimSpace(s)
}

func parseDate(s string) string {
	s = cleanField(s)
	// German: DD.MM.YY or DD.MM.YYYY
	if strings.Contains(s, ".") {
		formats := []string{"02.01.2006", "02.01.06"}
		for _, f := range formats {
			if t, err := time.Parse(f, s); err == nil {
				return t.Format("2006-01-02")
			}
		}
	}
	// ISO: YYYY-MM-DD
	if strings.Contains(s, "-") {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			return t.Format("2006-01-02")
		}
	}
	// US: MM/DD/YYYY
	if strings.Contains(s, "/") {
		formats := []string{"01/02/2006", "01/02/06"}
		for _, f := range formats {
			if t, err := time.Parse(f, s); err == nil {
				return t.Format("2006-01-02")
			}
		}
	}
	return ""
}

func parseAmount(s string, isGerman bool) (float64, error) {
	s = cleanField(s)
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}

	if isGerman {
		// German: 1.300,00 → 1300.00  or  -42,50 → -42.50
		// Remove thousand-sep dots, replace comma with dot
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else {
		// Already "standard" or US: just remove commas
		s = strings.ReplaceAll(s, ",", "")
	}

	return strconv.ParseFloat(s, 64)
}

func detectDelimiter(file multipart.File) rune {
	buf := make([]byte, 2048)
	n, _ := file.Read(buf)
	sample := string(buf[:n])
	if seeker, ok := file.(io.ReadSeeker); ok {
		seeker.Seek(0, io.SeekStart)
	}
	if strings.Count(sample, ";") > strings.Count(sample, ",") {
		return ';'
	}
	return ','
}

// findDefaultCategory returns the ID of a default category for imports
func (h *Handler) findDefaultCategory(isIncome bool) string {
	cats, err := h.DB.GetCategories(&isIncome)
	if err != nil || len(cats) == 0 {
		return ""
	}
	// Prefer "Sonstige Einnahmen" / "Sonstige Ausgaben"
	needle := "Sonstige Ausgaben"
	if isIncome {
		needle = "Sonstige Einnahmen"
	}
	for _, c := range cats {
		if strings.Contains(c.Name, needle) {
			return c.ID
		}
	}
	return cats[0].ID // fallback: first category of that type
}
