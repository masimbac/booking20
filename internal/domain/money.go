package domain

// Money is an ISO 4217 amount (amount as decimal string per OpenAPI).
type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}
