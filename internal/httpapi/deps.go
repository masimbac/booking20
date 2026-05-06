package httpapi

import (
	"github.com/parama/booking/internal/app/bookings"
	"github.com/parama/booking/internal/app/catalog"
	"github.com/parama/booking/internal/app/conversations"
	"github.com/parama/booking/internal/app/customers"
	"github.com/parama/booking/internal/app/notifications"
	"github.com/parama/booking/internal/app/payments"
	"github.com/parama/booking/internal/app/scheduling"
	"github.com/parama/booking/internal/app/tenancy"
)

// Deps wires application services into HTTP handlers (Phase 3+).
type Deps struct {
	Tenancy               *tenancy.Application
	Catalog               *catalog.Application
	Customers             *customers.Application
	Scheduling            *scheduling.Application
	Bookings              *bookings.Application
	Payments              *payments.Application
	Conversations         *conversations.Application
	Notifications         *notifications.Application
	PlatformAPIKey        string
	RequirePlatformAPIKey bool
	SkipTenantCheck       bool
}
