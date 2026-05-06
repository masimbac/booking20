package domain

import "time"

// ConversationChannel identifies the chat surface (OpenAPI enum).
type ConversationChannel string

const (
	ChannelWhatsApp ConversationChannel = "whatsapp"
)

// Conversation ties a customer to an async messaging thread.
type Conversation struct {
	ID               string                 `json:"id"`
	BusinessID       string                 `json:"business_id"`
	CustomerID       string                 `json:"customer_id"`
	Channel          ConversationChannel    `json:"channel"`
	ProviderThreadID string                 `json:"provider_thread_id,omitempty"`
	State            string                 `json:"state"`
	Context          map[string]any         `json:"-"`
	LastActivityAt   time.Time              `json:"last_activity_at"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}
