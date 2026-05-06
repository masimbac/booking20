package domain

import "time"

// Customer is a CRM record scoped to one business.
type Customer struct {
	ID             string         `json:"id"`
	BusinessID     string         `json:"business_id"`
	PhoneE164      string         `json:"phone_e164"`
	DisplayName    string         `json:"display_name,omitempty"`
	Preferences    map[string]any `json:"preferences,omitempty"`
	MarketingOptIn bool           `json:"marketing_opt_in"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}
