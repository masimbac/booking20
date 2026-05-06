package customers

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/parama/booking/internal/app/ports"
	"github.com/parama/booking/internal/domain"
)

// Application orchestrates CRM customer use cases.
type Application struct {
	Customers ports.CustomerRepository
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

// CreateCustomerInput mirrors POST /customers.
type CreateCustomerInput struct {
	BusinessID     string
	PhoneE164      string
	DisplayName    string
	Preferences    map[string]any
	MarketingOptIn *bool
}

// CreateCustomer persists a new customer.
func (a *Application) CreateCustomer(ctx context.Context, in CreateCustomerInput) (*domain.Customer, error) {
	if strings.TrimSpace(in.BusinessID) == "" {
		return nil, fmt.Errorf("%w: business_id is required", domain.ErrInvalid)
	}
	phone := strings.TrimSpace(in.PhoneE164)
	if phone == "" {
		return nil, fmt.Errorf("%w: phone_e164 is required", domain.ErrInvalid)
	}
	t := a.now()
	id := ulid.MustNew(ulid.Timestamp(t), ulid.Monotonic(rand.Reader, 0)).String()
	optIn := false
	if in.MarketingOptIn != nil {
		optIn = *in.MarketingOptIn
	}
	c := &domain.Customer{
		ID:             id,
		BusinessID:     strings.TrimSpace(in.BusinessID),
		PhoneE164:      phone,
		DisplayName:    strings.TrimSpace(in.DisplayName),
		Preferences:    in.Preferences,
		MarketingOptIn: optIn,
		CreatedAt:      t,
		UpdatedAt:      t,
	}
	if err := a.Customers.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListCustomers wraps the repository.
func (a *Application) ListCustomers(ctx context.Context, businessID string, opt ports.ListCustomersOptions) ([]domain.Customer, string, error) {
	return a.Customers.List(ctx, businessID, opt)
}

// GetCustomer returns one customer by id.
func (a *Application) GetCustomer(ctx context.Context, businessID, customerID string) (*domain.Customer, error) {
	return a.Customers.Get(ctx, businessID, customerID)
}

// GetCustomerByPhone resolves a customer by E.164 phone.
func (a *Application) GetCustomerByPhone(ctx context.Context, businessID, phoneE164 string) (*domain.Customer, error) {
	phone := strings.TrimSpace(phoneE164)
	if phone == "" {
		return nil, fmt.Errorf("%w: phone query is required", domain.ErrInvalid)
	}
	return a.Customers.GetByPhone(ctx, businessID, phone)
}

// PatchCustomerInput mirrors PATCH /customers/{id}.
type PatchCustomerInput struct {
	PhoneE164      *string
	DisplayName    *string
	Preferences    map[string]any
	MarketingOptIn *bool
}

// PatchCustomer applies partial update.
func (a *Application) PatchCustomer(ctx context.Context, businessID, customerID string, patch PatchCustomerInput) (*domain.Customer, error) {
	c, err := a.Customers.Get(ctx, businessID, customerID)
	if err != nil {
		return nil, err
	}
	if patch.PhoneE164 != nil {
		next := strings.TrimSpace(*patch.PhoneE164)
		if next == "" {
			return nil, fmt.Errorf("%w: phone_e164 cannot be empty", domain.ErrInvalid)
		}
		if next != c.PhoneE164 {
			other, err := a.Customers.GetByPhone(ctx, businessID, next)
			if err != nil {
				if !errors.Is(err, domain.ErrNotFound) {
					return nil, err
				}
			} else if other.ID != c.ID {
				return nil, fmt.Errorf("%w: phone already registered for this business", domain.ErrConflict)
			}
			c.PhoneE164 = next
		}
	}
	if patch.DisplayName != nil {
		c.DisplayName = strings.TrimSpace(*patch.DisplayName)
	}
	if patch.Preferences != nil {
		c.Preferences = patch.Preferences
	}
	if patch.MarketingOptIn != nil {
		c.MarketingOptIn = *patch.MarketingOptIn
	}
	c.UpdatedAt = a.now()
	if err := a.Customers.Save(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}
