package tenancy

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

// Application orchestrates tenant lifecycle use cases.
type Application struct {
	Businesses ports.BusinessRepository
	Now        func() time.Time
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

// RegisterBusinessInput mirrors POST /platform/businesses.
type RegisterBusinessInput struct {
	Name      string
	LegalName string
	Timezone  string
	Contact   map[string]any
}

// RegisterBusiness creates a new tenant.
func (a *Application) RegisterBusiness(ctx context.Context, in RegisterBusinessInput) (*domain.Business, error) {
	if err := validateBusinessIn(in); err != nil {
		return nil, err
	}
	now := a.now()
	id := ulid.MustNew(ulid.Timestamp(now), ulid.Monotonic(rand.Reader, 0)).String()
	b := &domain.Business{
		ID:        id,
		Name:      strings.TrimSpace(in.Name),
		LegalName: strings.TrimSpace(in.LegalName),
		Timezone:  strings.TrimSpace(in.Timezone),
		Contact:   in.Contact,
		Status:    domain.BusinessActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := a.Businesses.Create(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func validateBusinessIn(in RegisterBusinessInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("%w: name is required", domain.ErrInvalid)
	}
	if strings.TrimSpace(in.Timezone) == "" {
		return fmt.Errorf("%w: timezone is required", domain.ErrInvalid)
	}
	return nil
}

// PatchBusinessInput mirrors PATCH /businesses/{id}.
type PatchBusinessInput struct {
	Name      *string
	LegalName *string
	Timezone  *string
	Contact   map[string]any
	Status    *domain.BusinessStatus
}

// GetBusiness returns a tenant by id.
func (a *Application) GetBusiness(ctx context.Context, id string) (*domain.Business, error) {
	return a.Businesses.Get(ctx, id)
}

// PatchBusiness applies partial updates.
func (a *Application) PatchBusiness(ctx context.Context, id string, patch PatchBusinessInput) (*domain.Business, error) {
	b, err := a.Businesses.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	now := a.now()
	if patch.Name != nil {
		if strings.TrimSpace(*patch.Name) == "" {
			return nil, fmt.Errorf("%w: name cannot be empty", domain.ErrInvalid)
		}
		b.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.LegalName != nil {
		b.LegalName = strings.TrimSpace(*patch.LegalName)
	}
	if patch.Timezone != nil {
		if strings.TrimSpace(*patch.Timezone) == "" {
			return nil, fmt.Errorf("%w: timezone cannot be empty", domain.ErrInvalid)
		}
		b.Timezone = strings.TrimSpace(*patch.Timezone)
	}
	if patch.Contact != nil {
		b.Contact = patch.Contact
	}
	if patch.Status != nil {
		b.Status = *patch.Status
	}
	b.UpdatedAt = now
	if err := a.Businesses.Save(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}
