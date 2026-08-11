package db

import (
	"database/sql"
	"fmt"

	"github.com/ChancellorCookie/fulla/internal/core"
)

// --- Accounts ---

func (db *DB) CreateAccount(r core.CreateAccountRequest) (core.Account, error) {
	a := core.Account{
		ID:        NewID(),
		Name:      r.Name,
		Type:      r.Type,
		Currency:  r.Currency,
		CreatedAt: Now(),
		UpdatedAt: Now(),
	}
	if a.Currency == "" {
		a.Currency = "EUR"
	}
	_, err := db.conn.Exec(
		"INSERT INTO accounts (id, name, type, balance, currency, created_at, updated_at) VALUES (?,?,?,?,?,?,?)",
		a.ID, a.Name, a.Type, a.Balance, a.Currency, a.CreatedAt, a.UpdatedAt,
	)
	return a, err
}

func (db *DB) GetAccounts() ([]core.Account, error) {
	rows, err := db.conn.Query("SELECT id, name, type, balance, currency, created_at, updated_at FROM accounts ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAccounts(rows)
}

func (db *DB) GetAccount(id string) (core.Account, error) {
	row := db.conn.QueryRow("SELECT id, name, type, balance, currency, created_at, updated_at FROM accounts WHERE id=?", id)
	var a core.Account
	err := row.Scan(&a.ID, &a.Name, &a.Type, &a.Balance, &a.Currency, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (db *DB) UpdateAccountBalance(id string, delta float64) error {
	_, err := db.conn.Exec("UPDATE accounts SET balance = balance + ?, updated_at = ? WHERE id = ?", delta, Now(), id)
	return err
}

func (db *DB) DeleteAccount(id string) error {
	_, err := db.conn.Exec("DELETE FROM accounts WHERE id = ?", id)
	return err
}

// RecalcAccountBalance recomputes balance from all posted transactions
func (db *DB) RecalcAccountBalance(id string) error {
	var sum sql.NullFloat64
	err := db.conn.QueryRow("SELECT SUM(amount) FROM transactions WHERE account_id = ? AND status = 'posted'", id).Scan(&sum)
	if err != nil {
		return err
	}
	balance := 0.0
	if sum.Valid {
		balance = sum.Float64
	}
	_, err = db.conn.Exec("UPDATE accounts SET balance = ?, updated_at = ? WHERE id = ?", balance, Now(), id)
	return err
}

func scanAccounts(rows *sql.Rows) ([]core.Account, error) {
	var out []core.Account
	for rows.Next() {
		var a core.Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Balance, &a.Currency, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return out, fmt.Errorf("scan account: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
