package db

import (
	"fmt"

	"github.com/ChancellorCookie/fulla/internal/core"
)

// GetForecast returns expected income/expenses grouped by category with individual items
func (db *DB) GetForecast(month string) (*core.ForecastResponse, error) {
	rows, err := db.conn.Query(
		`SELECT rt.id, rt.category_id, c.name, c.color, rt.description, rt.amount
		 FROM recurring_transactions rt
		 JOIN categories c ON c.id = rt.category_id
		 WHERE rt.active = 1
		 ORDER BY c.name, rt.description`)
	if err != nil { return nil, err }
	defer rows.Close()

	type item struct{ id, catID, catName, color, desc string; amt float64 }
	var items []item
	for rows.Next() {
		var i item
		if err := rows.Scan(&i.id, &i.catID, &i.catName, &i.color, &i.desc, &i.amt); err != nil { continue }
		items = append(items, i)
	}

	catMap := map[string]*core.CategoryForecast{}
	var order []string
	var income, expenses float64
	for _, i := range items {
		cf, ok := catMap[i.catID]
		if !ok {
			cf = &core.CategoryForecast{CategoryID: i.catID, CategoryName: i.catName, Color: i.color}
			catMap[i.catID] = cf
			order = append(order, i.catID)
		}
		cf.Items = append(cf.Items, core.ForecastItem{ID: i.id, Name: i.desc, Amount: i.amt})
		cf.Total += i.amt
		if i.amt > 0 { income += i.amt } else { expenses += i.amt }
	}

	byCat := make([]core.CategoryForecast, 0, len(order))
	for _, id := range order { byCat = append(byCat, *catMap[id]) }

	return &core.ForecastResponse{
		Month: month, Income: income, Expenses: expenses,
		Balance: income + expenses, ByCat: byCat,
	}, rows.Err()
}

func (db *DB) CreateTransaction(r core.CreateTransactionRequest) (core.Transaction, error) {
	return db.CreateTransactionWithStatus(r, "posted")
}

func (db *DB) CreateTransactionWithStatus(r core.CreateTransactionRequest, status string) (core.Transaction, error) {
	t := core.Transaction{
		ID: NewID(), AccountID: r.AccountID, CategoryID: r.CategoryID,
		Amount: r.Amount, Description: r.Description, Date: r.Date,
		Note: r.Note, Status: status, CreatedAt: Now(),
	}
	if t.Status == "" { t.Status = "posted" }
	_, err := db.conn.Exec(
		`INSERT INTO transactions (id, account_id, category_id, amount, description, date, note, status, recurring_match_id, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.AccountID, t.CategoryID, t.Amount, t.Description, t.Date, t.Note, t.Status, r.RecurringMatchID, t.CreatedAt,
	)
	if err != nil { return t, err }
	if t.Status == "posted" { _ = db.UpdateAccountBalance(t.AccountID, t.Amount) }
	return t, nil
}

func (db *DB) GetTransactions(accountID, month, status string, limit int) ([]core.Transaction, error) {
	query := "SELECT id, account_id, category_id, amount, description, date, note, status, recurring_match_id, created_at FROM transactions"
	var args []any
	var conditions []string
	if accountID != "" { conditions = append(conditions, "account_id = ?"); args = append(args, accountID) }
	if month != "" { conditions = append(conditions, "date LIKE ?"); args = append(args, month+"%") }
	if status != "" { conditions = append(conditions, "status = ?"); args = append(args, status) }
	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, c := range conditions[1:] { query += " AND " + c }
	}
	query += " ORDER BY date DESC, created_at DESC"
	if limit > 0 { query += " LIMIT ?"; args = append(args, limit) }

	rows, err := db.conn.Query(query, args...)
	if err != nil { return nil, err }
	defer rows.Close()

	var out = make([]core.Transaction, 0)
	for rows.Next() {
		var t core.Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.CategoryID, &t.Amount, &t.Description, &t.Date, &t.Note, &t.Status, &t.RecurringMatchID, &t.CreatedAt); err != nil {
			return out, fmt.Errorf("scan transaction: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (db *DB) BulkUpdateTransactions(req core.BulkUpdateRequest) (int, error) {
	if len(req.IDs) == 0 { return 0, nil }

	placeholders := make([]string, len(req.IDs))
	args := make([]any, 0)
	for i, id := range req.IDs { placeholders[i] = "?"; args = append(args, id) }

	sets := []string{}
	setArgs := []any{}
	if req.CategoryID != "" { sets = append(sets, "category_id = ?"); setArgs = append(setArgs, req.CategoryID) }
	if req.Status != "" { sets = append(sets, "status = ?"); setArgs = append(setArgs, req.Status) }

	var balanceUpdates []struct{ acctID string; amt float64 }
	if req.Status == "posted" {
		query := fmt.Sprintf(
			"SELECT account_id, amount FROM transactions WHERE id IN (%s) AND status = 'pending'",
			joinPlaceholders(len(req.IDs)),
		)
		rows, err := db.conn.Query(query, strsToAnys(req.IDs)...)
		if err == nil {
			for rows.Next() {
				var acctID string; var amt float64
				if err := rows.Scan(&acctID, &amt); err == nil {
					balanceUpdates = append(balanceUpdates, struct{ acctID string; amt float64 }{acctID, amt})
				}
			}
			rows.Close()
		}
	}

	if len(sets) == 0 { return 0, nil }
	allArgs := append(setArgs, strsToAnys(req.IDs)...)
	query := fmt.Sprintf("UPDATE transactions SET %s WHERE id IN (%s)", joinSet(sets), joinPlaceholders(len(req.IDs)))
	result, err := db.conn.Exec(query, allArgs...)
	if err != nil { return 0, err }

	for _, u := range balanceUpdates { _ = db.UpdateAccountBalance(u.acctID, u.amt) }
	n, _ := result.RowsAffected()
	return int(n), nil
}

func (db *DB) DeleteTransaction(id string) error {
	var amount float64
	var accountID, status string
	err := db.conn.QueryRow("SELECT amount, account_id, status FROM transactions WHERE id=?", id).Scan(&amount, &accountID, &status)
	if err != nil { return err }
	_, err = db.conn.Exec("DELETE FROM transactions WHERE id=?", id)
	if err != nil { return err }
	if status == "posted" { _ = db.UpdateAccountBalance(accountID, -amount) }
	return nil
}

func (db *DB) ToggleExcludeTransaction(id string) error {
	var current string
	err := db.conn.QueryRow("SELECT recurring_match_id FROM transactions WHERE id = ?", id).Scan(&current)
	if err != nil { return err }
	val := "manual"
	if current != "" { val = "" }
	_, err = db.conn.Exec("UPDATE transactions SET recurring_match_id = ? WHERE id = ?", val, id)
	return err
}

func (db *DB) UpdateTransaction(id string, r core.CreateTransactionRequest) error {
	_, err := db.conn.Exec(
		`UPDATE transactions SET category_id=?, amount=?, description=?, date=?, note=?, recurring_match_id=? WHERE id=?`,
		r.CategoryID, r.Amount, r.Description, r.Date, r.Note, r.RecurringMatchID, id)
	return err
}

func (db *DB) GetMonthSummary(accountID, month string) (*core.MonthSummary, error) {
	query := `SELECT COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0),
	                 COALESCE(SUM(CASE WHEN amount < 0 THEN amount ELSE 0 END), 0)
	          FROM transactions WHERE date LIKE ? AND status = 'posted'`
	args := []any{month + "%"}
	if accountID != "" { query += " AND account_id = ?"; args = append(args, accountID) }

	s := &core.MonthSummary{Month: month}
	err := db.conn.QueryRow(query, args...).Scan(&s.Income, &s.Expenses)
	if err != nil { return nil, err }
	s.Balance = s.Income + s.Expenses

	catQuery := `SELECT t.category_id, c.name, c.color, SUM(t.amount) as total, COUNT(*) as cnt
	             FROM transactions t JOIN categories c ON c.id = t.category_id
	             WHERE t.date LIKE ? AND t.status = 'posted' AND t.recurring_match_id = ''`
	catArgs := []any{month + "%"}
	if accountID != "" { catQuery += " AND t.account_id = ?"; catArgs = append(catArgs, accountID) }
	catQuery += " GROUP BY t.category_id ORDER BY total"

	rows, err := db.conn.Query(catQuery, catArgs...)
	if err != nil { return nil, err }
	defer rows.Close()

	for rows.Next() {
		var cs core.CategorySummary
		if err := rows.Scan(&cs.CategoryID, &cs.CategoryName, &cs.Color, &cs.Amount, &cs.Count); err != nil {
			return s, fmt.Errorf("scan cat summary: %w", err)
		}
		s.ByCategory = append(s.ByCategory, cs)
	}
	return s, rows.Err()
}

func joinPlaceholders(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 { s += "," }
		s += "?"
	}
	return s
}

func joinSet(parts []string) string {
	s := ""
	for i, p := range parts {
		if i > 0 { s += ", " }
		s += p
	}
	return s
}

func strsToAnys(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss { out[i] = s }
	return out
}
