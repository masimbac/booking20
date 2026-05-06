package ports

import (
	"context"

	"github.com/parama/booking/internal/domain"
)

// ListServicesOptions controls query and pagination for services.
type ListServicesOptions struct {
	Limit       int32
	Cursor      string
	ActiveOnly  bool
}

// ListStaffOptions controls query and pagination for staff.
type ListStaffOptions struct {
	Limit  int32
	Cursor string
}

// ServiceRepository persists catalog services (Dynamo SK prefix SERVICE#).
type ServiceRepository interface {
	Create(ctx context.Context, s *domain.CatalogService) error
	Get(ctx context.Context, businessID, serviceID string) (*domain.CatalogService, error)
	List(ctx context.Context, businessID string, opt ListServicesOptions) ([]domain.CatalogService, string, error)
	Save(ctx context.Context, s *domain.CatalogService) error // full replace (patch)
	Delete(ctx context.Context, businessID, serviceID string) error
}

// StaffRepository persists staff (Dynamo SK prefix STAFF#).
type StaffRepository interface {
	Create(ctx context.Context, s *domain.Staff) error
	Get(ctx context.Context, businessID, staffID string) (*domain.Staff, error)
	List(ctx context.Context, businessID string, opt ListStaffOptions) ([]domain.Staff, string, error)
	Save(ctx context.Context, s *domain.Staff) error
	Delete(ctx context.Context, businessID, staffID string) error
}
