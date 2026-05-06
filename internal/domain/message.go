package domain

import "time"

// MessageDirection matches OpenAPI.
type MessageDirection string

const (
	MessageInbound  MessageDirection = "inbound"
	MessageOutbound MessageDirection = "outbound"
)

// Message is one turn in a conversation (stored under PK CONVO#).
type Message struct {
	ID                string                 `json:"id"`
	ConversationID    string                 `json:"conversation_id"`
	Direction         MessageDirection       `json:"direction"`
	Body              string                 `json:"body"`
	Structured        map[string]any         `json:"structured,omitempty"`
	ProviderMessageID string                 `json:"provider_message_id,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
}
