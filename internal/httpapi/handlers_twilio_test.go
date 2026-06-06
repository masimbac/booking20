package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/parama/booking/internal/app/twilio"
)

func TestPostTwilioWebhook_OK(t *testing.T) {
	app := &twilio.Application{}
	deps := &Deps{Twilio: app}
	r := NewRouter(RouterConfig{Phase: "9"}, deps)

	form := url.Values{
		"MessageSid": {"SM123"},
		"From":       {"+15551234567"},
		"To":         {"+15557654321"},
		"Body":       {"hello"},
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/twilio", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(context.Background())

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
}

func TestPostTwilioWebhook_notWired(t *testing.T) {
	r := NewRouter(RouterConfig{Phase: "9"}, &Deps{})
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/twilio", strings.NewReader("MessageSid=SM1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status: %d", rec.Code)
	}
}
