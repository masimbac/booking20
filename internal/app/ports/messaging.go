package ports

import (
	"context"

	"github.com/parama/booking/internal/domain"
)

// ListMessagesOptions paginates message history for a conversation.
type ListMessagesOptions struct {
	Limit  int32
	Cursor string
}

// MessagingRepository stores conversations (per business), index rows, messages (per CONVO#), and webhook dedup markers.
type MessagingRepository interface {
	GetConversation(ctx context.Context, businessID, conversationID string) (*domain.Conversation, error)
	GetConversationIDByCustomerChannel(ctx context.Context, businessID, customerID string, channel domain.ConversationChannel) (string, error)
	CreateConversationWithIndex(ctx context.Context, c *domain.Conversation) error
	SaveConversation(ctx context.Context, c *domain.Conversation) error

	AppendMessage(ctx context.Context, m *domain.Message) error
	ListMessages(ctx context.Context, conversationID string, opt ListMessagesOptions) ([]domain.Message, string, error)

	// TryAcquireWebhookDedup inserts a deduplication key; returns false if this webhook delivery was already processed.
	TryAcquireWebhookDedup(ctx context.Context, businessID, provider, providerMessageID string) (bool, error)
}
