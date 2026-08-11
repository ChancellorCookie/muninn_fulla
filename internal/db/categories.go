package db

import (
	"fmt"

	"github.com/ChancellorCookie/fulla/internal/core"
)

// --- Categories ---

func (db *DB) CreateCategory(r core.CreateCategoryRequest) (core.Category, error) {
	c := core.Category{
		ID:        NewID(),
		Name:      r.Name,
		Color:     r.Color,
		Icon:      r.Icon,
		IsIncome:  r.IsIncome,
		CreatedAt: Now(),
	}
	if c.Color == "" {
		c.Color = "#888888"
	}
	_, err := db.conn.Exec(
		"INSERT INTO categories (id, name, color, icon, is_income, created_at) VALUES (?,?,?,?,?,?)",
		c.ID, c.Name, c.Color, c.Icon, boolToInt(c.IsIncome), c.CreatedAt,
	)
	return c, err
}

func (db *DB) GetCategories(income *bool) ([]core.Category, error) {
	query := "SELECT id, name, color, icon, is_income, created_at FROM categories"
	var args []any
	if income != nil {
		query += " WHERE is_income = ?"
		if *income {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	query += " ORDER BY name"
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []core.Category
	for rows.Next() {
		var c core.Category
		var isIncome int
		if err := rows.Scan(&c.ID, &c.Name, &c.Color, &c.Icon, &isIncome, &c.CreatedAt); err != nil {
			return out, fmt.Errorf("scan category: %w", err)
		}
		c.IsIncome = isIncome != 0
		out = append(out, c)
	}
	return out, rows.Err()
}

func (db *DB) DeleteCategory(id string) error {
	_, err := db.conn.Exec("DELETE FROM categories WHERE id = ?", id)
	return err
}

func (db *DB) UpdateCategory(id string, name, color string) error {
	_, err := db.conn.Exec("UPDATE categories SET name = ?, color = ? WHERE id = ?", name, color, id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
