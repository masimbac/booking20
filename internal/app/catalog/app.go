package catalog

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

// Application orchestrates catalog use cases (services + staff).
type Application struct {
	Services ports.ServiceRepository
	Staff    ports.StaffRepository
	Now      func() time.Time
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

// CreateServiceInput mirrors POST services.
type CreateServiceInput struct {
	BusinessID      string
	Name            string
	DurationMinutes int
	Price           *domain.Money
	Active          *bool
	Metadata        map[string]any
}

// CreateService persists a new offering.
func (a *Application) CreateService(ctx context.Context, in CreateServiceInput) (*domain.CatalogService, error) {
	if err := validateServiceBasics(in.Name, in.DurationMinutes); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.BusinessID) == "" {
		return nil, fmt.Errorf("%w: business_id is required", domain.ErrInvalid)
	}
	t := a.now()
	id := ulid.MustNew(ulid.Timestamp(t), ulid.Monotonic(rand.Reader, 0)).String()
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	s := &domain.CatalogService{
		ID:              id,
		BusinessID:      strings.TrimSpace(in.BusinessID),
		Name:            strings.TrimSpace(in.Name),
		DurationMinutes: in.DurationMinutes,
		Price:           in.Price,
		Active:          active,
		Metadata:        in.Metadata,
		CreatedAt:       t,
		UpdatedAt:       t,
	}
	if err := a.Services.Create(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func validateServiceBasics(name string, duration int) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: name is required", domain.ErrInvalid)
	}
	if duration < 1 {
		return fmt.Errorf("%w: duration_minutes must be >= 1", domain.ErrInvalid)
	}
	return nil
}

// ListServices wraps the repository.
func (a *Application) ListServices(ctx context.Context, businessID string, opt ports.ListServicesOptions) ([]domain.CatalogService, string, error) {
	return a.Services.List(ctx, businessID, opt)
}

// GetService returns one offering.
func (a *Application) GetService(ctx context.Context, businessID, serviceID string) (*domain.CatalogService, error) {
	return a.Services.Get(ctx, businessID, serviceID)
}

// PatchServiceInput mirrors PATCH service.
type PatchServiceInput struct {
	Name            *string
	DurationMinutes *int
	Price           *domain.Money
	Active          *bool
	Metadata        map[string]any
}

// PatchService applies partial update.
func (a *Application) PatchService(ctx context.Context, businessID, serviceID string, patch PatchServiceInput) (*domain.CatalogService, error) {
	s, err := a.Services.Get(ctx, businessID, serviceID)
	if err != nil {
		return nil, err
	}
	if patch.Name != nil {
		if strings.TrimSpace(*patch.Name) == "" {
			return nil, fmt.Errorf("%w: name cannot be empty", domain.ErrInvalid)
		}
		s.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.DurationMinutes != nil {
		if *patch.DurationMinutes < 1 {
			return nil, fmt.Errorf("%w: duration_minutes must be >= 1", domain.ErrInvalid)
		}
		s.DurationMinutes = *patch.DurationMinutes
	}
	if patch.Price != nil {
		s.Price = patch.Price
	}
	if patch.Active != nil {
		s.Active = *patch.Active
	}
	if patch.Metadata != nil {
		s.Metadata = patch.Metadata
	}
	s.UpdatedAt = a.now()
	if err := a.Services.Save(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

// DeleteService removes a service row.
func (a *Application) DeleteService(ctx context.Context, businessID, serviceID string) error {
	return a.Services.Delete(ctx, businessID, serviceID)
}

// CreateStaffInput mirrors POST staff.
type CreateStaffInput struct {
	BusinessID   string
	DisplayName  string
	Role         string
	ServiceIDs   []string
	Active       *bool
	Metadata     map[string]any
}

// CreateStaff persists staff.
func (a *Application) CreateStaff(ctx context.Context, in CreateStaffInput) (*domain.Staff, error) {
	if strings.TrimSpace(in.DisplayName) == "" {
		return nil, fmt.Errorf("%w: display_name is required", domain.ErrInvalid)
	}
	if strings.TrimSpace(in.BusinessID) == "" {
		return nil, fmt.Errorf("%w: business_id is required", domain.ErrInvalid)
	}
	t := a.now()
	id := ulid.MustNew(ulid.Timestamp(t), ulid.Monotonic(rand.Reader, 0)).String()
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	st := &domain.Staff{
		ID:          id,
		BusinessID:  strings.TrimSpace(in.BusinessID),
		DisplayName: strings.TrimSpace(in.DisplayName),
		Role:        strings.TrimSpace(in.Role),
		ServiceIDs:  append([]string(nil), in.ServiceIDs...),
		Active:      active,
		Metadata:    in.Metadata,
		CreatedAt:   t,
		UpdatedAt:   t,
	}
	if err := a.Staff.Create(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

// ListStaff wraps the repository.
func (a *Application) ListStaff(ctx context.Context, businessID string, opt ports.ListStaffOptions) ([]domain.Staff, string, error) {
	return a.Staff.List(ctx, businessID, opt)
}

// GetStaff returns one staff member.
func (a *Application) GetStaff(ctx context.Context, businessID, staffID string) (*domain.Staff, error) {
	return a.Staff.Get(ctx, businessID, staffID)
}

// PatchStaffInput mirrors PATCH staff.
type PatchStaffInput struct {
	DisplayName *string
	Role        *string
	ServiceIDs  *[]string // nil = unchanged; non-nil (possibly empty) replaces the full list
	Active      *bool
	Metadata    map[string]any
}

// PatchStaff applies partial update.
func (a *Application) PatchStaff(ctx context.Context, businessID, staffID string, patch PatchStaffInput) (*domain.Staff, error) {
	st, err := a.Staff.Get(ctx, businessID, staffID)
	if err != nil {
		return nil, err
	}
	if patch.DisplayName != nil {
		if strings.TrimSpace(*patch.DisplayName) == "" {
			return nil, fmt.Errorf("%w: display_name cannot be empty", domain.ErrInvalid)
		}
		st.DisplayName = strings.TrimSpace(*patch.DisplayName)
	}
	if patch.Role != nil {
		st.Role = strings.TrimSpace(*patch.Role)
	}
	if patch.ServiceIDs != nil {
		st.ServiceIDs = append([]string(nil), (*patch.ServiceIDs)...)
	}
	if patch.Active != nil {
		st.Active = *patch.Active
	}
	if patch.Metadata != nil {
		st.Metadata = patch.Metadata
	}
	st.UpdatedAt = a.now()
	if err := a.Staff.Save(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

// DeleteStaff removes a staff row.
func (a *Application) DeleteStaff(ctx context.Context, businessID, staffID string) error {
	return a.Staff.Delete(ctx, businessID, staffID)
}
