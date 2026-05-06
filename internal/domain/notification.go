package domain

import "time"

// NotificationKind matches openapi NotificationKind.
type NotificationKind string

const (
	NotificationReminder     NotificationKind = "reminder"
	NotificationConfirmation NotificationKind = "confirmation"
	NotificationPromo        NotificationKind = "promo"
)

// NotificationStatus matches openapi NotificationStatus.
type NotificationStatus string

const (
	NotificationScheduled NotificationStatus = "scheduled"
	NotificationSent      NotificationStatus = "sent"
	NotificationFailed    NotificationStatus = "failed"
)

// Notification is a scheduled outbound message scoped to one business.
type Notification struct {
	ID           string               `json:"id"`
	BusinessID   string               `json:"business_id"`
	Kind         NotificationKind     `json:"kind"`
	Channel      string               `json:"channel"`
	Status       NotificationStatus   `json:"status"`
	ScheduledAt  time.Time            `json:"scheduled_at"`
	CustomerID   string               `json:"customer_id,omitempty"`
	BookingID    string               `json:"booking_id,omitempty"`
	Payload      map[string]any       `json:"payload,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}
