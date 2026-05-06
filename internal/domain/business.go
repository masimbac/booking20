package domain

import "time"

// BusinessStatus is the tenant lifecycle flag.
type BusinessStatus string

const (
	BusinessActive    BusinessStatus = "active"
	BusinessSuspended BusinessStatus = "suspended"
)

// Business is the tenant aggregate root (no AWS/HTTP imports).
type Business struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	LegalName string                 `json:"legal_name,omitempty"`
	Timezone  string                 `json:"timezone"`
	Contact   map[string]any         `json:"contact,omitempty"`
	Status    BusinessStatus         `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}
