package domain

import "time"

// Staff is a staff member belonging to a business.
type Staff struct {
	ID          string         `json:"id"`
	BusinessID  string         `json:"business_id"`
	DisplayName string         `json:"display_name"`
	Role        string         `json:"role,omitempty"`
	ServiceIDs  []string       `json:"service_ids,omitempty"`
	Active      bool           `json:"active"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
