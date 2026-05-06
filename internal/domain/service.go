package domain

import "time"

// CatalogService is an offering within a business (name avoids collision with the word "service" in layers).
type CatalogService struct {
	ID              string         `json:"id"`
	BusinessID      string         `json:"business_id"`
	Name            string         `json:"name"`
	DurationMinutes int            `json:"duration_minutes"`
	Price           *Money         `json:"price,omitempty"`
	Active          bool           `json:"active"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
