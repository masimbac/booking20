package ports

import (
	"context"
	"time"

	"github.com/parama/booking/internal/domain"
)

// ListNotificationsOptions paginates notifications under a business partition.
type ListNotificationsOptions struct {
	Status string // scheduled | sent | failed | "" for all
	Limit  int32
	Cursor string
}

// NotificationRepository persists notifications (GSI4 due queue when status is scheduled).
type NotificationRepository interface {
	Create(ctx context.Context, n *domain.Notification) error
	Get(ctx context.Context, businessID, notificationID string) (*domain.Notification, error)
	List(ctx context.Context, businessID string, opt ListNotificationsOptions) ([]domain.Notification, string, error)
	ListDueScheduled(ctx context.Context, beforeUTC time.Time, limit int32) ([]domain.Notification, error)
	Save(ctx context.Context, n *domain.Notification) error
}
