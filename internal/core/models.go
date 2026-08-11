package core

// Account represents a financial account (Giro, Spar, Depot, etc.)
type Account struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"` // checking, savings, investment
	Balance   float64 `json:"balance"`
	Currency  string  `json:"currency"` // default: EUR
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// Category groups transactions (Gehalt, Auto, Versicherung, etc.)
type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`     // hex color
	Icon      string `json:"icon"`      // emoji or icon name
	IsIncome  bool   `json:"is_income"` // true for income categories
	CreatedAt string `json:"created_at"`
}

// Transaction is a single income or expense entry
type Transaction struct {
	ID               string  `json:"id"`
	AccountID        string  `json:"account_id"`
	CategoryID       string  `json:"category_id"`
	Amount           float64 `json:"amount"`
	Description      string  `json:"description"`
	Date             string  `json:"date"`
	Note             string  `json:"note"`
	Status           string  `json:"status"`            // "pending" or "posted"
	RecurringMatchID string  `json:"recurring_match_id"` // linked recurring if auto-matched
	CreatedAt        string  `json:"created_at"`
}

// BulkUpdateRequest for batch-editing transactions
type BulkUpdateRequest struct {
	IDs        []string `json:"ids"`
	CategoryID string   `json:"category_id,omitempty"`
	Status     string   `json:"status,omitempty"`
}

// RecurringTransaction is a template for recurring bookings
type RecurringTransaction struct {
	ID           string  `json:"id"`
	AccountID    string  `json:"account_id"`
	CategoryID   string  `json:"category_id"`
	Amount       float64 `json:"amount"`
	Description  string  `json:"description"`
	IntervalKind string  `json:"interval_kind"` // monthly, quarterly, yearly
	IntervalN    int     `json:"interval_n"`    // every N intervals (default 1)
	NextDue      string  `json:"next_due"`      // ISO 8601 date
	Active       bool    `json:"active"`
	CreatedAt    string  `json:"created_at"`
}

// ForecastItem is one recurring transaction in a forecast
type ForecastItem struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

// CategoryForecast groups forecast items by category
type CategoryForecast struct {
	CategoryID   string          `json:"category_id"`
	CategoryName string          `json:"category_name"`
	Color        string          `json:"color"`
	Total        float64         `json:"total"`
	Items        []ForecastItem  `json:"items"`
}

// ForecastResponse for /api/forecast
type ForecastResponse struct {
	Month    string             `json:"month"`
	Income   float64            `json:"income"`
	Expenses float64            `json:"expenses"`
	Balance  float64            `json:"balance"`
	ByCat    []CategoryForecast `json:"by_cat"`
}

// HealthResponse for /api/health
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Now     string `json:"now"`
}

// --- Request types ---

type CreateAccountRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Currency string `json:"currency,omitempty"`
}

type CreateCategoryRequest struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	Icon     string `json:"icon,omitempty"`
	IsIncome bool   `json:"is_income"`
}

type CreateTransactionRequest struct {
	AccountID        string  `json:"account_id"`
	CategoryID       string  `json:"category_id"`
	Amount           float64 `json:"amount"`
	Description      string  `json:"description"`
	Date             string  `json:"date"`
	Note             string  `json:"note,omitempty"`
	Status           string  `json:"status,omitempty"`
	RecurringMatchID string  `json:"recurring_match_id,omitempty"`
}

type CreateRecurringRequest struct {
	AccountID    string  `json:"account_id"`
	CategoryID   string  `json:"category_id"`
	Amount       float64 `json:"amount"`
	Description  string  `json:"description"`
	IntervalKind string  `json:"interval_kind"`
	IntervalN    int     `json:"interval_n,omitempty"`
	NextDue      string  `json:"next_due"`
}

// --- Dashboard / overview types ---

type MonthSummary struct {
	Month       string            `json:"month"` // YYYY-MM
	Income      float64           `json:"income"`
	Expenses    float64           `json:"expenses"`
	Balance     float64           `json:"balance"`
	ByCategory  []CategorySummary `json:"by_category"`
}

type CategorySummary struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Color        string  `json:"color"`
	Amount       float64 `json:"amount"`
	Count        int     `json:"count"`
}
