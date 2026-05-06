package ports

import (
	"context"

	"github.com/parama/booking/internal/domain"
)

// BusinessRepository persists tenants (Dynamo PK BUSINESS#, SK META#).
type BusinessRepository interface {
	Create(ctx context.Context, b *domain.Business) error
	Get(ctx context.Context, id string) (*domain.Business, error)
	Save(ctx context.Context, b *domain.Business) error
}
