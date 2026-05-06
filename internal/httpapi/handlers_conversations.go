package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/parama/booking/internal/app/conversations"
	"github.com/parama/booking/internal/app/ports"
	"github.com/parama/booking/internal/domain"
)

type ensureConversationReq struct {
	CustomerID       string `json:"customer_id"`
	Channel          string `json:"channel"`
	ProviderThreadID string `json:"provider_thread_id,omitempty"`
}

func (d *Deps) postEnsureConversation(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	var body ensureConversationReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	ch := domain.ConversationChannel(strings.TrimSpace(strings.ToLower(body.Channel)))
	if ch != domain.ChannelWhatsApp {
		WriteUnprocessable(w, r, "channel must be whatsapp")
		return
	}
	out, created, err := d.Conversations.EnsureConversation(r.Context(), conversations.EnsureConversationInput{
		BusinessID:       bid,
		CustomerID:       body.CustomerID,
		Channel:          ch,
		ProviderThreadID: body.ProviderThreadID,
	})
	if mapAppErr(w, r, err) {
		return
	}
	if created {
		writeJSON(w, http.StatusCreated, out)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (d *Deps) getConversationDoc(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	cid := chi.URLParam(r, "conversationId")
	out, err := d.Conversations.GetConversation(r.Context(), bid, cid)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (d *Deps) listConversationMessages(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	cid := chi.URLParam(r, "conversationId")
	items, next, err := d.Conversations.ListMessages(r.Context(), bid, cid, ports.ListMessagesOptions{
		Limit:  parseLimit32(r, 20, 100),
		Cursor: r.URL.Query().Get("cursor"),
	})
	if mapAppErr(w, r, err) {
		return
	}
	resp := map[string]any{"items": items}
	if next != "" {
		resp["next_cursor"] = next
	}
	writeJSON(w, http.StatusOK, resp)
}

type postConversationMessageReq struct {
	Body       string         `json:"body"`
	Structured map[string]any `json:"structured,omitempty"`
}

func (d *Deps) postConversationMessage(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	cid := chi.URLParam(r, "conversationId")
	var body postConversationMessageReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	out, err := d.Conversations.CreateOutboundMessage(r.Context(), bid, cid, conversations.CreateOutboundMessageInput{
		Body:       body.Body,
		Structured: body.Structured,
	})
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
