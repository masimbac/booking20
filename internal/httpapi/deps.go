package httpapi

import (
	"github.com/parama/booking/internal/app/bookings"
	"github.com/parama/booking/internal/app/catalog"
	"github.com/parama/booking/internal/app/customers"
	"github.com/parama/booking/internal/app/scheduling"
	"github.com/parama/booking/internal/app/tenancy"
)

// Deps wires application services into HTTP handlers (Phase 3+).
type Deps struct {
	Tenancy         *tenancy.Application
	Catalog         *catalog.Application
	Customers       *customers.Application
	Scheduling      *scheduling.Application
	Bookings        *bookings.Application
	PlatformAPIKey  string
	SkipTenantCheck bool
}
