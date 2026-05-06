package ports

import (
	"context"
	"time"

	"github.com/parama/booking/internal/domain"
)

// ListCustomersOptions paginates customer listing under a business partition.
type ListCustomersOptions struct {
	Limit  int32
	Cursor string
}

// CustomerRepository persists customers (SK prefix CUSTOMER#, GSI2 phone lookup).
type CustomerRepository interface {
	Create(ctx context.Context, c *domain.Customer) error
	Get(ctx context.Context, businessID, customerID string) (*domain.Customer, error)
	GetByPhone(ctx context.Context, businessID, phoneE164 string) (*domain.Customer, error)
	List(ctx context.Context, businessID string, opt ListCustomersOptions) ([]domain.Customer, string, error)
	Save(ctx context.Context, c *domain.Customer) error
}

// AvailabilityRepository stores merged weekly rules for a business (single item AVAIL#RULES).
type AvailabilityRepository interface {
	PutRules(ctx context.Context, businessID string, rules []domain.AvailabilityRule, updatedAt time.Time) error
	GetRules(ctx context.Context, businessID string) ([]domain.AvailabilityRule, time.Time, error)
}
