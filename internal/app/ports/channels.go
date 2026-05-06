package ports

import "context"

// OutboundWhatsAppInput is a normalized outbound text send.
type OutboundWhatsAppInput struct {
	BusinessID string
	CustomerID string
	ToE164     string
	Body       string
}

// ChannelOutbound sends messages to external chat providers.
type ChannelOutbound interface {
	SendWhatsAppText(ctx context.Context, in OutboundWhatsAppInput) error
}
