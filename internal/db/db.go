package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite connection
type DB struct {
	conn *sql.DB
}

// New opens (or creates) the SQLite database and runs migrations
func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(1) // SQLite works best with a single writer

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// Close shuts down the database
func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS accounts (
		id         TEXT PRIMARY KEY,
		name       TEXT NOT NULL,
		type       TEXT NOT NULL DEFAULT 'checking',
		balance    REAL NOT NULL DEFAULT 0,
		currency   TEXT NOT NULL DEFAULT 'EUR',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS categories (
		id        TEXT PRIMARY KEY,
		name      TEXT NOT NULL,
		color     TEXT NOT NULL DEFAULT '#888888',
		icon      TEXT NOT NULL DEFAULT '',
		is_income INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id          TEXT PRIMARY KEY,
		account_id  TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
		category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE SET NULL,
		amount      REAL NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		date        TEXT NOT NULL,
		note        TEXT NOT NULL DEFAULT '',
		status      TEXT NOT NULL DEFAULT 'posted',
		recurring_match_id TEXT NOT NULL DEFAULT '',
		created_at  TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_tx_date ON transactions(date);
	CREATE INDEX IF NOT EXISTS idx_tx_account ON transactions(account_id);

	CREATE TABLE IF NOT EXISTS recurring_transactions (
		id            TEXT PRIMARY KEY,
		account_id    TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
		category_id   TEXT NOT NULL REFERENCES categories(id) ON DELETE SET NULL,
		amount        REAL NOT NULL,
		description   TEXT NOT NULL DEFAULT '',
		interval_kind TEXT NOT NULL DEFAULT 'monthly',
		interval_n    INTEGER NOT NULL DEFAULT 1,
		next_due      TEXT NOT NULL,
		active        INTEGER NOT NULL DEFAULT 1,
		created_at    TEXT NOT NULL
	);
	`
	_, err := db.conn.Exec(schema)
	return err
}

// Now returns ISO 8601 timestamp
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}
