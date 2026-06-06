package twilio

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/parama/booking/internal/domain"
)

func TestVerifySignature_valid(t *testing.T) {
	const token = "test-auth-token"
	form := url.Values{
		"Body":       {"hello"},
		"From":       {"+15551234567"},
		"MessageSid": {"SM123"},
		"To":         {"+15557654321"},
	}
	webhookURL := "https://example.com/v1/webhooks/twilio"
	sig := computeSignature(token, webhookURL, form)

	app := &Application{AuthToken: token}
	if !app.VerifySignature(webhookURL, form, sig) {
		t.Fatal("expected valid signature")
	}
}

func TestVerifySignature_invalid(t *testing.T) {
	app := &Application{AuthToken: "secret"}
	form := url.Values{"Body": {"hello"}}
	if app.VerifySignature("https://example.com/v1/webhooks/twilio", form, "bad-signature") {
		t.Fatal("expected invalid signature")
	}
}

func TestVerifySignature_skippedWhenTokenUnset(t *testing.T) {
	app := &Application{}
	if !app.VerifySignature("https://example.com", url.Values{}, "") {
		t.Fatal("expected verification to pass when auth token unset")
	}
}

func TestHandleInbound_logsMessage(t *testing.T) {
	app := &Application{}
	form := url.Values{
		"MessageSid": {"SM999"},
		"AccountSid": {"AC123"},
		"From":       {"+15551234567"},
		"To":         {"+15557654321"},
		"Body":       {"book haircut"},
	}
	if err := app.HandleInbound(context.Background(), "https://example.com/v1/webhooks/twilio", form, ""); err != nil {
		t.Fatalf("HandleInbound: %v", err)
	}
}

func TestHandleInbound_requiresMessageSid(t *testing.T) {
	app := &Application{}
	err := app.HandleInbound(context.Background(), "https://example.com/v1/webhooks/twilio", url.Values{}, "")
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestHandleInbound_rejectsBadSignature(t *testing.T) {
	app := &Application{AuthToken: "secret"}
	form := url.Values{"MessageSid": {"SM1"}, "Body": {"hi"}}
	err := app.HandleInbound(context.Background(), "https://example.com/v1/webhooks/twilio", form, "bad")
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}
