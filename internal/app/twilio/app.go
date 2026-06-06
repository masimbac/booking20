package twilio

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strings"

	"github.com/parama/booking/internal/domain"
)

// Application handles inbound Twilio webhooks (SMS / WhatsApp via Twilio).
type Application struct {
	AuthToken string // optional: X-Twilio-Signature verification when set
}

// InboundMessage is the subset of Twilio form fields we care about for inbound SMS.
type InboundMessage struct {
	MessageSID string
	AccountSID string
	From       string
	To         string
	Body       string
}

// VerifySignature checks Twilio's X-Twilio-Signature when AuthToken is configured.
func (a *Application) VerifySignature(webhookURL string, form url.Values, header string) bool {
	token := strings.TrimSpace(a.AuthToken)
	if token == "" {
		return true
	}
	sig := strings.TrimSpace(header)
	if sig == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(sig), []byte(computeSignature(token, webhookURL, form))) == 1
}

// HandleInbound parses a verified Twilio webhook payload and logs the message.
func (a *Application) HandleInbound(ctx context.Context, webhookURL string, form url.Values, signature string) error {
	_ = ctx
	if !a.VerifySignature(webhookURL, form, signature) {
		return fmt.Errorf("%w: invalid twilio webhook signature", domain.ErrInvalid)
	}
	msg := parseInbound(form)
	if strings.TrimSpace(msg.MessageSID) == "" {
		return fmt.Errorf("%w: MessageSid is required", domain.ErrInvalid)
	}
	slog.Info("twilio_inbound_message",
		"message_sid", msg.MessageSID,
		"account_sid", msg.AccountSID,
		"from", msg.From,
		"to", msg.To,
		"body", msg.Body,
	)
	return nil
}

func parseInbound(form url.Values) InboundMessage {
	return InboundMessage{
		MessageSID: strings.TrimSpace(form.Get("MessageSid")),
		AccountSID: strings.TrimSpace(form.Get("AccountSid")),
		From:       strings.TrimSpace(form.Get("From")),
		To:         strings.TrimSpace(form.Get("To")),
		Body:       strings.TrimSpace(form.Get("Body")),
	}
}

func computeSignature(authToken, webhookURL string, form url.Values) string {
	var keys []string
	for k := range form {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(webhookURL)
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(form.Get(k))
	}

	mac := hmac.New(sha1.New, []byte(authToken))
	_, _ = mac.Write([]byte(b.String()))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
