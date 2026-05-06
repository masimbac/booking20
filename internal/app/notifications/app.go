package notifications

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/parama/booking/internal/app/ports"
	"github.com/parama/booking/internal/domain"
)

// Application schedules and dispatches notifications.
type Application struct {
	Repo      ports.NotificationRepository
	Customers ports.CustomerRepository
	Outbound  ports.ChannelOutbound
	Now       func() time.Time
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

// CreateNotificationInput mirrors POST /notifications.
type CreateNotificationInput struct {
	BusinessID  string
	Kind        domain.NotificationKind
	Channel     string
	ScheduledAt time.Time
	CustomerID  string
	BookingID   string
	Payload     map[string]any
}

// Create schedules a notification.
func (a *Application) Create(ctx context.Context, in CreateNotificationInput) (*domain.Notification, error) {
	if strings.TrimSpace(in.BusinessID) == "" {
		return nil, fmt.Errorf("%w: business_id is required", domain.ErrInvalid)
	}
	ch := strings.TrimSpace(strings.ToLower(in.Channel))
	if ch == "" {
		return nil, fmt.Errorf("%w: channel is required", domain.ErrInvalid)
	}
	switch in.Kind {
	case domain.NotificationReminder, domain.NotificationConfirmation, domain.NotificationPromo:
	default:
		return nil, fmt.Errorf("%w: invalid notification kind", domain.ErrInvalid)
	}
	if ch == "whatsapp" && strings.TrimSpace(in.CustomerID) == "" {
		return nil, fmt.Errorf("%w: customer_id is required for whatsapp notifications", domain.ErrInvalid)
	}
	t := a.now()
	id := ulid.MustNew(ulid.Timestamp(t), ulid.Monotonic(rand.Reader, 0)).String()
	n := &domain.Notification{
		ID:          id,
		BusinessID:  strings.TrimSpace(in.BusinessID),
		Kind:        in.Kind,
		Channel:     ch,
		Status:      domain.NotificationScheduled,
		ScheduledAt: in.ScheduledAt.UTC(),
		CustomerID:  strings.TrimSpace(in.CustomerID),
		BookingID:   strings.TrimSpace(in.BookingID),
		Payload:     in.Payload,
		CreatedAt:   t,
		UpdatedAt:   t,
	}
	if err := a.Repo.Create(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

// List returns notifications for a tenant.
func (a *Application) List(ctx context.Context, businessID string, status string, limit int32, cursor string) ([]domain.Notification, string, error) {
	return a.Repo.List(ctx, businessID, ports.ListNotificationsOptions{
		Status: status,
		Limit:  limit,
		Cursor: cursor,
	})
}

// DispatchDue finds due scheduled notifications, sends WhatsApp where applicable, updates status.
func (a *Application) DispatchDue(ctx context.Context, limit int32) (dispatched int, _ error) {
	if a.Outbound == nil || a.Customers == nil {
		return 0, nil
	}
	candidates, err := a.Repo.ListDueScheduled(ctx, a.now(), limit)
	if err != nil {
		return 0, err
	}
	for i := range candidates {
		n := &candidates[i]
		fresh, err := a.Repo.Get(ctx, n.BusinessID, n.ID)
		if err != nil {
			continue
		}
		if fresh.Status != domain.NotificationScheduled {
			continue
		}
		if err := a.dispatchOne(ctx, fresh); err != nil {
			t := a.now()
			fresh.Status = domain.NotificationFailed
			fresh.UpdatedAt = t
			_ = a.Repo.Save(ctx, fresh)
			continue
		}
		t := a.now()
		fresh.Status = domain.NotificationSent
		fresh.UpdatedAt = t
		if err := a.Repo.Save(ctx, fresh); err != nil {
			return dispatched, err
		}
		dispatched++
	}
	return dispatched, nil
}

func (a *Application) dispatchOne(ctx context.Context, n *domain.Notification) error {
	switch strings.ToLower(strings.TrimSpace(n.Channel)) {
	case "whatsapp":
	default:
		return fmt.Errorf("%w: unsupported channel %q", domain.ErrInvalid, n.Channel)
	}
	if strings.TrimSpace(n.CustomerID) == "" {
		return fmt.Errorf("%w: customer_id is required", domain.ErrInvalid)
	}
	cust, err := a.Customers.Get(ctx, n.BusinessID, n.CustomerID)
	if err != nil {
		return err
	}
	body := outboundBodyFor(n)
	return a.Outbound.SendWhatsAppText(ctx, ports.OutboundWhatsAppInput{
		BusinessID: n.BusinessID,
		CustomerID: n.CustomerID,
		ToE164:     cust.PhoneE164,
		Body:       body,
	})
}

func outboundBodyFor(n *domain.Notification) string {
	if n.Payload != nil {
		if s, _ := n.Payload["body"].(string); strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	switch n.Kind {
	case domain.NotificationReminder:
		if n.Payload != nil {
			if start, ok := n.Payload["start_at"].(string); ok && start != "" {
				return fmt.Sprintf("Reminder: your booking starts at %s.", start)
			}
		}
		return "Reminder: you have an upcoming appointment with us."
	case domain.NotificationConfirmation:
		return "Your booking is confirmed. Thank you!"
	case domain.NotificationPromo:
		return "We have an offer for you — reply for details."
	default:
		return "You have a notification from your booking."
	}
}

// ScheduleBookingReminder is invoked when a booking is first created (EventSink hook from cmd/api).
func (a *Application) ScheduleBookingReminder(ctx context.Context, b *domain.Booking) error {
	if a.Repo == nil || b == nil {
		return nil
	}
	reminderAt := b.StartAt.Add(-24 * time.Hour)
	if !reminderAt.After(a.now()) {
		reminderAt = a.now().Add(2 * time.Minute)
	}
	payload := map[string]any{
		"booking_id": b.ID,
		"start_at":   b.StartAt.UTC().Format(time.RFC3339),
	}
	_, err := a.Create(ctx, CreateNotificationInput{
		BusinessID:  b.BusinessID,
		Kind:        domain.NotificationReminder,
		Channel:     "whatsapp",
		ScheduledAt: reminderAt,
		CustomerID:  b.CustomerID,
		BookingID:   b.ID,
		Payload:     payload,
	})
	return err
}
