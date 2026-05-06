package conversations

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/parama/booking/internal/app/bookings"
	"github.com/parama/booking/internal/app/customers"
	"github.com/parama/booking/internal/app/ports"
	"github.com/parama/booking/internal/app/tenancy"
	"github.com/parama/booking/internal/domain"
)

const (
	stateIdle = "idle"
)

// Application coordinates conversations, messages, and chat-driven use cases.
type Application struct {
	Messaging         ports.MessagingRepository
	Customers         *customers.Application
	Bookings          *bookings.Application
	Tenancy           *tenancy.Application
	Outbound          ports.ChannelOutbound
	Now               func() time.Time
	WhatsAppAppSecret string // optional: Meta X-Hub-Signature-256 verification
}

func (a *Application) now() time.Time {
	if a.Now != nil {
		t := a.Now()
		if !t.IsZero() {
			return t
		}
	}
	return time.Now().UTC()
}

func (a *Application) outbound() ports.ChannelOutbound {
	return a.Outbound
}

// EnsureConversationInput mirrors POST /conversations.
type EnsureConversationInput struct {
	BusinessID       string
	CustomerID       string
	Channel          domain.ConversationChannel
	ProviderThreadID string
}

// EnsureConversation get-or-creates the thread for (business, customer, channel).
func (a *Application) EnsureConversation(ctx context.Context, in EnsureConversationInput) (*domain.Conversation, bool, error) {
	if strings.TrimSpace(in.BusinessID) == "" || strings.TrimSpace(in.CustomerID) == "" {
		return nil, false, fmt.Errorf("%w: business_id and customer_id are required", domain.ErrInvalid)
	}
	if in.Channel == "" {
		return nil, false, fmt.Errorf("%w: channel is required", domain.ErrInvalid)
	}
	if _, err := a.Customers.GetCustomer(ctx, in.BusinessID, in.CustomerID); err != nil {
		return nil, false, err
	}
	if id, err := a.Messaging.GetConversationIDByCustomerChannel(ctx, in.BusinessID, in.CustomerID, in.Channel); err == nil && id != "" {
		c, err := a.Messaging.GetConversation(ctx, in.BusinessID, id)
		return c, false, err
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, false, err
	}
	t := a.now()
	cid := ulid.MustNew(ulid.Timestamp(t), ulid.Monotonic(rand.Reader, 0)).String()
	c := &domain.Conversation{
		ID:               cid,
		BusinessID:       strings.TrimSpace(in.BusinessID),
		CustomerID:       strings.TrimSpace(in.CustomerID),
		Channel:          in.Channel,
		ProviderThreadID: strings.TrimSpace(in.ProviderThreadID),
		State:            stateIdle,
		Context:          map[string]any{},
		LastActivityAt:   t,
		CreatedAt:        t,
		UpdatedAt:        t,
	}
	if err := a.Messaging.CreateConversationWithIndex(ctx, c); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			id, e2 := a.Messaging.GetConversationIDByCustomerChannel(ctx, in.BusinessID, in.CustomerID, in.Channel)
			if e2 == nil && id != "" {
				existing, e3 := a.Messaging.GetConversation(ctx, in.BusinessID, id)
				return existing, false, e3
			}
		}
		return nil, false, err
	}
	return c, true, nil
}

// GetConversation loads metadata after verifying business scope.
func (a *Application) GetConversation(ctx context.Context, businessID, conversationID string) (*domain.Conversation, error) {
	c, err := a.Messaging.GetConversation(ctx, businessID, conversationID)
	if err != nil {
		return nil, err
	}
	if c.BusinessID != businessID {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

// ListMessages returns paginated history for a conversation in this business.
func (a *Application) ListMessages(ctx context.Context, businessID, conversationID string, opt ports.ListMessagesOptions) ([]domain.Message, string, error) {
	if _, err := a.GetConversation(ctx, businessID, conversationID); err != nil {
		return nil, "", err
	}
	return a.Messaging.ListMessages(ctx, conversationID, opt)
}

// CreateOutboundMessageInput mirrors POST messages.
type CreateOutboundMessageInput struct {
	Body       string
	Structured map[string]any
}

// CreateOutboundMessage appends an outbound turn (admin/system).
func (a *Application) CreateOutboundMessage(ctx context.Context, businessID, conversationID string, in CreateOutboundMessageInput) (*domain.Message, error) {
	c, err := a.GetConversation(ctx, businessID, conversationID)
	if err != nil {
		return nil, err
	}
	body := strings.TrimSpace(in.Body)
	if body == "" {
		return nil, fmt.Errorf("%w: body is required", domain.ErrInvalid)
	}
	t := a.now()
	mid := ulid.MustNew(ulid.Timestamp(t), ulid.Monotonic(rand.Reader, 0)).String()
	m := &domain.Message{
		ID:             mid,
		ConversationID: conversationID,
		Direction:      domain.MessageOutbound,
		Body:           body,
		Structured:     in.Structured,
		CreatedAt:      t,
	}
	if err := a.Messaging.AppendMessage(ctx, m); err != nil {
		return nil, err
	}
	c.LastActivityAt = t
	c.UpdatedAt = t
	if err := a.Messaging.SaveConversation(ctx, c); err != nil {
		return nil, err
	}
	if ob := a.outbound(); ob != nil {
		cust, err := a.Customers.GetCustomer(ctx, businessID, c.CustomerID)
		if err == nil && cust != nil {
			_ = ob.SendWhatsAppText(ctx, ports.OutboundWhatsAppInput{
				BusinessID: businessID,
				CustomerID: c.CustomerID,
				ToE164:     cust.PhoneE164,
				Body:       body,
			})
		}
	}
	return m, nil
}

// AppendInboundMessage records a customer/provider inbound message (no outbound).
func (a *Application) AppendInboundMessage(ctx context.Context, businessID string, c *domain.Conversation, body, providerMsgID string, structured map[string]any) (*domain.Message, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("%w: message body is empty", domain.ErrInvalid)
	}
	t := a.now()
	mid := ulid.MustNew(ulid.Timestamp(t), ulid.Monotonic(rand.Reader, 0)).String()
	m := &domain.Message{
		ID:                mid,
		ConversationID:    c.ID,
		Direction:         domain.MessageInbound,
		Body:              body,
		Structured:        structured,
		ProviderMessageID: providerMsgID,
		CreatedAt:         t,
	}
	if err := a.Messaging.AppendMessage(ctx, m); err != nil {
		return nil, err
	}
	c.LastActivityAt = t
	c.UpdatedAt = t
	if err := a.Messaging.SaveConversation(ctx, c); err != nil {
		return nil, err
	}
	return m, nil
}

// WhatsAppNormalizedPayload is the canonical JSON shape for /webhooks/whatsapp.
type WhatsAppNormalizedPayload struct {
	BusinessID string `json:"business_id"`
	FromE164   string `json:"from_e164"`
	Text       string `json:"text"`
	MessageID  string `json:"message_id"`
}

// VerifyWhatsAppSignature checks Meta-style X-Hub-Signature-256 when WhatsAppAppSecret is set.
func (a *Application) VerifyWhatsAppSignature(body []byte, header string) bool {
	secret := strings.TrimSpace(a.WhatsAppAppSecret)
	if secret == "" {
		return true
	}
	sig := strings.TrimSpace(header)
	if !strings.HasPrefix(sig, "sha256=") {
		return false
	}
	wantHex := strings.TrimPrefix(sig, "sha256=")
	want, err := hex.DecodeString(wantHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	got := mac.Sum(nil)
	return subtle.ConstantTimeCompare(got, want) == 1
}

// HandleWhatsAppWebhook processes a verified inbound provider message.
func (a *Application) HandleWhatsAppWebhook(ctx context.Context, body []byte) error {
	var p WhatsAppNormalizedPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return fmt.Errorf("invalid webhook json: %w", domain.ErrInvalid)
	}
	if strings.TrimSpace(p.BusinessID) == "" || strings.TrimSpace(p.FromE164) == "" {
		return fmt.Errorf("%w: business_id and from_e164 are required", domain.ErrInvalid)
	}
	if a.Tenancy != nil {
		if _, err := a.Tenancy.GetBusiness(ctx, p.BusinessID); err != nil {
			return err
		}
	}
	ok, err := a.Messaging.TryAcquireWebhookDedup(ctx, p.BusinessID, "whatsapp", p.MessageID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	cust, err := a.Customers.GetCustomerByPhone(ctx, p.BusinessID, p.FromE164)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		cust, err = a.Customers.CreateCustomer(ctx, customers.CreateCustomerInput{
			BusinessID:  p.BusinessID,
			PhoneE164:   p.FromE164,
			DisplayName: "WhatsApp",
		})
		if err != nil {
			return err
		}
	}
	conv, _, err := a.EnsureConversation(ctx, EnsureConversationInput{
		BusinessID: p.BusinessID,
		CustomerID: cust.ID,
		Channel:    domain.ChannelWhatsApp,
	})
	if err != nil {
		return err
	}
	if strings.TrimSpace(p.Text) == "" {
		return nil
	}
	if _, err := a.AppendInboundMessage(ctx, p.BusinessID, conv, p.Text, p.MessageID, nil); err != nil {
		return err
	}
	return a.dispatchInbound(ctx, p.BusinessID, conv, cust, strings.TrimSpace(p.Text), p.MessageID)
}

func (a *Application) dispatchInbound(ctx context.Context, businessID string, conv *domain.Conversation, cust *domain.Customer, text, providerMsgID string) error {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return nil
	}
	cmd := strings.ToUpper(fields[0])
	switch cmd {
	case "HELP":
		return a.reply(ctx, businessID, conv, cust, "Commands: HELP, BOOK <serviceId> <staffId> <start RFC3339>")
	case "BOOK":
		if a.Bookings == nil {
			return a.reply(ctx, businessID, conv, cust, "Bookings are not available.")
		}
		if len(fields) < 4 {
			return a.reply(ctx, businessID, conv, cust, "Usage: BOOK <service_id> <staff_id> <start_at RFC33>")
		}
		svcID, stfID, startS := fields[1], fields[2], fields[3]
		startAt, err := time.Parse(time.RFC3339, startS)
		if err != nil {
			return a.reply(ctx, businessID, conv, cust, "Invalid start time; use RFC3339 (e.g. 2026-05-07T15:00:00Z)")
		}
		idem := "wa-webhook"
		if providerMsgID != "" {
			idem = "wa-webhook-" + providerMsgID
		} else {
			idem = idem + "-" + ulid.MustNew(ulid.Timestamp(a.now()), ulid.Monotonic(rand.Reader, 0)).String()
		}
		b, _, err := a.Bookings.CreateBooking(ctx, idem, bookings.CreateBookingInput{
			BusinessID: businessID,
			CustomerID: cust.ID,
			ServiceID:  svcID,
			StaffID:    stfID,
			StartAt:    startAt,
		})
		if err != nil {
			if errors.Is(err, domain.ErrConflict) {
				return a.reply(ctx, businessID, conv, cust, "Could not book (slot conflict or invalid state). Try another time.")
			}
			return a.reply(ctx, businessID, conv, cust, "Booking failed: try again or use the API.")
		}
		msg := fmt.Sprintf("Booked: id=%s status=%s start=%s", b.ID, b.Status, b.StartAt.UTC().Format(time.RFC3339))
		return a.reply(ctx, businessID, conv, cust, msg)
	default:
		return a.reply(ctx, businessID, conv, cust, "Thanks — send HELP for commands.")
	}
}

func (a *Application) reply(ctx context.Context, businessID string, conv *domain.Conversation, cust *domain.Customer, body string) error {
	t := a.now()
	mid := ulid.MustNew(ulid.Timestamp(t), ulid.Monotonic(rand.Reader, 0)).String()
	m := &domain.Message{
		ID:             mid,
		ConversationID: conv.ID,
		Direction:      domain.MessageOutbound,
		Body:           body,
		CreatedAt:      t,
	}
	if err := a.Messaging.AppendMessage(ctx, m); err != nil {
		return err
	}
	conv.LastActivityAt = t
	conv.UpdatedAt = t
	if err := a.Messaging.SaveConversation(ctx, conv); err != nil {
		return err
	}
	if ob := a.outbound(); ob != nil {
		return ob.SendWhatsAppText(ctx, ports.OutboundWhatsAppInput{
			BusinessID: businessID,
			CustomerID: cust.ID,
			ToE164:     cust.PhoneE164,
			Body:       body,
		})
	}
	return nil
}

// HookHTTPStatus maps errors to provider-facing status codes (challenge handler pattern).
func HookHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, domain.ErrInvalid) {
		return http.StatusBadRequest
	}
	if errors.Is(err, domain.ErrNotFound) {
		return http.StatusBadRequest
	}
	return http.StatusBadRequest
}
