package httpapi

import (
	"github.com/parama/booking/internal/app/catalog"
	"github.com/parama/booking/internal/app/tenancy"
)

// Deps wires application services into HTTP handlers (Phase 3).
type Deps struct {
	Tenancy         *tenancy.Application
	Catalog         *catalog.Application
	PlatformAPIKey  string
	SkipTenantCheck bool
}
