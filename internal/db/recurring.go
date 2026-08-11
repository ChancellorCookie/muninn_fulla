package db

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/ChancellorCookie/fulla/internal/core"
)

// --- Recurring Transactions ---

func (db *DB) CreateRecurring(r core.CreateRecurringRequest) (core.RecurringTransaction, error) {
	n := r.IntervalN
	if n < 1 { n = 1 }
	nextDue := r.NextDue
	if nextDue == "" {
		// Auto-calculate from current date
		now := time.Now().UTC()
		nextDue = advanceDateStr(now.Format("2006-01-02"), r.IntervalKind, n)
	}
	rt := core.RecurringTransaction{
		ID:           NewID(),
		AccountID:    r.AccountID,
		CategoryID:   r.CategoryID,
		Amount:       r.Amount,
		Description:  r.Description,
		IntervalKind: r.IntervalKind,
		IntervalN:    n,
		NextDue:      nextDue,
		Active:       true,
		CreatedAt:    Now(),
	}
	_, err := db.conn.Exec(
		`INSERT INTO recurring_transactions (id, account_id, category_id, amount, description, interval_kind, interval_n, next_due, active, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		rt.ID, rt.AccountID, rt.CategoryID, rt.Amount, rt.Description,
		rt.IntervalKind, rt.IntervalN, rt.NextDue, boolToInt(rt.Active), rt.CreatedAt,
	)
	return rt, err
}

func (db *DB) GetRecurrings() ([]core.RecurringTransaction, error) {
	rows, err := db.conn.Query(
		`SELECT id, account_id, category_id, amount, description, interval_kind, interval_n, next_due, active, created_at
		 FROM recurring_transactions ORDER BY next_due`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []core.RecurringTransaction
	for rows.Next() {
		var rt core.RecurringTransaction
		var active int
		if err := rows.Scan(&rt.ID, &rt.AccountID, &rt.CategoryID, &rt.Amount,
			&rt.Description, &rt.IntervalKind, &rt.IntervalN, &rt.NextDue, &active, &rt.CreatedAt); err != nil {
			return out, fmt.Errorf("scan recurring: %w", err)
		}
		rt.Active = active != 0
		out = append(out, rt)
	}
	return out, rows.Err()
}

func (db *DB) ToggleRecurring(id string) error {
	_, err := db.conn.Exec("UPDATE recurring_transactions SET active = NOT active WHERE id = ?", id)
	return err
}

// GetRecurringHistory returns all posted transactions matching this recurring's pattern
// GetRecurringHistory returns all posted transactions matching this recurring's pattern
func (db *DB) GetRecurringHistory(rtID string) ([]core.Transaction, error) {
	var rt core.RecurringTransaction
	var active int
	err := db.conn.QueryRow(
		`SELECT id, account_id, category_id, amount, description, interval_kind, interval_n, next_due, active, created_at
		 FROM recurring_transactions WHERE id = ?`, rtID).Scan(
		&rt.ID, &rt.AccountID, &rt.CategoryID, &rt.Amount, &rt.Description,
		&rt.IntervalKind, &rt.IntervalN, &rt.NextDue, &active, &rt.CreatedAt)
	if err != nil {
		return make([]core.Transaction, 0), nil // not found → empty
	}

	// Find matching posted transactions
	tol := math.Abs(rt.Amount*0.01)
	if tol < 1 { tol = 1 }
	rows, err := db.conn.Query(
		`SELECT id, account_id, category_id, amount, description, date, note, status, created_at
		 FROM transactions WHERE status = 'posted' AND ABS(amount - ?) < ?
		 ORDER BY date DESC LIMIT 100`,
		rt.Amount, tol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rtNorm := normalizeDesc(rt.Description)
	var out = make([]core.Transaction, 0)
	for rows.Next() {
		var tx core.Transaction
		if err := rows.Scan(&tx.ID, &tx.AccountID, &tx.CategoryID, &tx.Amount,
			&tx.Description, &tx.Date, &tx.Note, &tx.Status, &tx.CreatedAt); err != nil {
			continue
		}
		if descriptionsMatch(normalizeDesc(tx.Description), rtNorm) {
			out = append(out, tx)
		}
	}
	return out, rows.Err()
}

func (db *DB) UpdateRecurring(id string, r core.CreateRecurringRequest) error {
	n := r.IntervalN
	if n < 1 { n = 1 }
	_, err := db.conn.Exec(
		`UPDATE recurring_transactions SET account_id=?, category_id=?, amount=?, description=?, interval_kind=?, interval_n=?, next_due=?
		 WHERE id=?`,
		r.AccountID, r.CategoryID, r.Amount, r.Description, r.IntervalKind, n, r.NextDue, id)
	return err
}

func (db *DB) DeleteRecurring(id string) error {
	_, err := db.conn.Exec("DELETE FROM recurring_transactions WHERE id = ?", id)
	return err
}

// normalizeDesc strips variable parts for fuzzy matching
func normalizeDesc(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	// Remove SWIFT prefixes
	for _, p := range []string{"svwz+", "eref+", "kref+", "mref+", "cred+", "abwa+"} {
		d = strings.ReplaceAll(d, p, "")
	}
	// Remove common date/number patterns
	d = regexp.MustCompile(`\d{2}\.\d{2}\.\d{2,4}`).ReplaceAllString(d, "")
	d = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`).ReplaceAllString(d, "")
	d = regexp.MustCompile(`\b\d{4,}\b`).ReplaceAllString(d, "")
	// Collapse whitespace
	d = strings.Join(strings.Fields(d), " ")
	return d
}

// descriptionsMatch checks if two normalized descriptions are similar enough
func descriptionsMatch(a, b string) bool {
	if a == "" || b == "" { return false }
	if strings.Contains(a, b) || strings.Contains(b, a) { return true }
	// Token overlap: at least 60% of shorter's tokens appear in longer
	aWords := strings.Fields(a)
	bWords := strings.Fields(b)
	if len(aWords) == 0 || len(bWords) == 0 { return false }
	shorter, longer := aWords, bWords
	if len(bWords) < len(aWords) { shorter, longer = bWords, aWords }
	matches := 0
	for _, w := range shorter {
		if len(w) < 3 { continue } // skip very short words
		for _, lw := range longer {
			if w == lw { matches++; break }
		}
	}
	threshold := int(float64(len(shorter)) * 0.6)
	return matches >= threshold
}

// FindMatchingRecurring finds an active recurring that matches description + amount
func (db *DB) FindMatchingRecurring(description string, amount float64) (*core.RecurringTransaction, error) {
	rows, err := db.conn.Query(
		`SELECT id, account_id, category_id, amount, description, interval_kind, interval_n, next_due, active, created_at
		 FROM recurring_transactions WHERE active = 1
		 ORDER BY next_due`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	normDesc := normalizeDesc(description)
	for rows.Next() {
		var rt core.RecurringTransaction
		var active int
		if err := rows.Scan(&rt.ID, &rt.AccountID, &rt.CategoryID, &rt.Amount,
			&rt.Description, &rt.IntervalKind, &rt.IntervalN, &rt.NextDue, &active, &rt.CreatedAt); err != nil {
			continue
		}
		rt.Active = active != 0

		// Amount: within 1% or 1€ (whichever is larger)
		tol := math.Abs(rt.Amount*0.01)
		if tol < 1 { tol = 1 }
		if math.Abs(rt.Amount-amount) > tol { continue }

		rtNorm := normalizeDesc(rt.Description)
		if descriptionsMatch(rtNorm, normDesc) {
			return &rt, nil
		}
	}
	return nil, nil
}

// MatchPendingToRecurring finds all pending transactions matching this recurring and auto-posts them
func (db *DB) MatchPendingToRecurring(rt *core.RecurringTransaction) (int, error) {
	// Find all pending transactions with approximate amount match
	tol := math.Abs(rt.Amount*0.01)
	if tol < 1 { tol = 1 }
	rows, err := db.conn.Query(
		`SELECT id, description FROM transactions WHERE status = 'pending' AND ABS(amount - ?) < ?`,
		rt.Amount, tol)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	rtLower := normalizeDesc(rt.Description)
	type match struct{ id string }
	var ids []string

	for rows.Next() {
		var id, desc string
		if err := rows.Scan(&id, &desc); err != nil { continue }
		if descriptionsMatch(normalizeDesc(desc), rtLower) {
			ids = append(ids, id)
		}
	}
	rows.Close() // MUST close before any Exec

	count := 0
	for _, id := range ids {
		var acctID string; var amt float64
		if err := db.conn.QueryRow("SELECT account_id, amount FROM transactions WHERE id = ?", id).Scan(&acctID, &amt); err != nil {
			continue
		}
		_, err := db.conn.Exec(
			"UPDATE transactions SET category_id = ?, status = 'posted' WHERE id = ?",
			rt.CategoryID, id)
		if err == nil {
			_ = db.UpdateAccountBalance(acctID, amt)
			count++
		}
	}
	return count, nil
}

// ProcessDueRecurrings creates transactions for all recurrings whose next_due <= today
func (db *DB) ProcessDueRecurrings() (int, error) {
	today := time.Now().UTC().Format("2006-01-02")

	rows, err := db.conn.Query(
		`SELECT id, account_id, category_id, amount, description, interval_kind, interval_n, next_due
		 FROM recurring_transactions WHERE active = 1 AND next_due <= ?`, today)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var rt core.RecurringTransaction
		if err := rows.Scan(&rt.ID, &rt.AccountID, &rt.CategoryID, &rt.Amount,
			&rt.Description, &rt.IntervalKind, &rt.IntervalN, &rt.NextDue); err != nil {
			return count, err
		}

		// Create the transaction
		tx := core.Transaction{
			ID:          NewID(),
			AccountID:   rt.AccountID,
			CategoryID:  rt.CategoryID,
			Amount:      rt.Amount,
			Description: rt.Description,
			Date:        rt.NextDue,
			CreatedAt:   Now(),
		}
		_, err := db.conn.Exec(
			`INSERT INTO transactions (id, account_id, category_id, amount, description, date, note, created_at)
			 VALUES (?,?,?,?,?,?,?,?)`,
			tx.ID, tx.AccountID, tx.CategoryID, tx.Amount, tx.Description, tx.Date, "", tx.CreatedAt,
		)
		if err != nil {
			return count, fmt.Errorf("create recurring tx: %w", err)
		}
		_ = db.UpdateAccountBalance(rt.AccountID, rt.Amount)

		// Advance next_due
		nextDue, err := advanceDate(rt.NextDue, rt.IntervalKind, rt.IntervalN)
		if err != nil {
			return count, fmt.Errorf("advance date: %w", err)
		}
		db.conn.Exec("UPDATE recurring_transactions SET next_due = ? WHERE id = ?", nextDue, rt.ID)
		count++
	}
	return count, rows.Err()
}

func advanceDate(dateStr, kind string, n int) (string, error) {
	d, err := time.Parse("2006-01-02", dateStr)
	if err != nil { return "", err }
	switch kind {
	case "monthly": d = d.AddDate(0, n, 0)
	case "quarterly": d = d.AddDate(0, 3*n, 0)
	case "yearly": d = d.AddDate(n, 0, 0)
	default: return "", fmt.Errorf("unknown interval kind: %s", kind)
	}
	return d.Format("2006-01-02"), nil
}

func advanceDateStr(dateStr, kind string, n int) string {
	s, _ := advanceDate(dateStr, kind, n)
	return s
}
