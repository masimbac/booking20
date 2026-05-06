package outbound

import (
	"context"

	"github.com/parama/booking/internal/app/ports"
)

// WhatsAppStub implements [ports.ChannelOutbound] as a no-op (replace with Meta/Twilio adapter).
type WhatsAppStub struct{}

func (WhatsAppStub) SendWhatsAppText(ctx context.Context, in ports.OutboundWhatsAppInput) error {
	_ = ctx
	_ = in
	return nil
}
