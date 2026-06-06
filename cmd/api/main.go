package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"

	"github.com/parama/booking/internal/adapters/dynamo"
	"github.com/parama/booking/internal/adapters/outbound"
	"github.com/parama/booking/internal/adapters/paymentstub"
	"github.com/parama/booking/internal/app/bookings"
	"github.com/parama/booking/internal/app/catalog"
	"github.com/parama/booking/internal/app/conversations"
	"github.com/parama/booking/internal/app/customers"
	"github.com/parama/booking/internal/app/notifications"
	"github.com/parama/booking/internal/app/payments"
	"github.com/parama/booking/internal/app/scheduling"
	"github.com/parama/booking/internal/app/tenancy"
	"github.com/parama/booking/internal/app/twilio"
	"github.com/parama/booking/internal/domain"
	"github.com/parama/booking/internal/httpapi"
	"github.com/parama/booking/internal/phase0"
)

type notifyOnBookingCreated struct {
	n *notifications.Application
}

func (w notifyOnBookingCreated) BookingCreated(ctx context.Context, b domain.Booking) error {
	if w.n == nil {
		return nil
	}
	return w.n.ScheduleBookingReminder(ctx, &b)
}

func (notifyOnBookingCreated) BookingLifecycle(ctx context.Context, b domain.Booking, transition string) error {
	_ = ctx
	_ = b
	_ = transition
	return nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx := context.Background()
	awscfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("aws config: %v", err)
	}
	table := os.Getenv("CORE_TABLE_NAME")
	if table == "" {
		log.Fatal("CORE_TABLE_NAME is required")
	}
	ddb := dynamodb.NewFromConfig(awscfg)

	bizRepo := &dynamo.BusinessRepository{Client: ddb, Table: table}
	svcRepo := &dynamo.ServiceRepository{Client: ddb, Table: table}
	stfRepo := &dynamo.StaffRepository{Client: ddb, Table: table}
	custRepo := &dynamo.CustomerRepository{Client: ddb, Table: table}
	availRepo := &dynamo.AvailabilityRepository{Client: ddb, Table: table}
	bookRepo := &dynamo.BookingRepository{Client: ddb, Table: table}
	msgRepo := &dynamo.MessagingRepository{Client: ddb, Table: table}
	payRepo := &dynamo.PaymentRepository{Client: ddb, Table: table}
	notifRepo := &dynamo.NotificationRepository{Client: ddb, Table: table}

	now := func() time.Time { return time.Now().UTC() }
	whatsappOut := outbound.WhatsAppStub{}
	notifApp := &notifications.Application{
		Repo:      notifRepo,
		Customers: custRepo,
		Outbound:  whatsappOut,
		Now:       now,
	}
	ten := &tenancy.Application{Businesses: bizRepo, Now: now}
	cat := &catalog.Application{Services: svcRepo, Staff: stfRepo, Now: now}
	crm := &customers.Application{Customers: custRepo, Now: now}
	sch := &scheduling.Application{
		Businesses: bizRepo,
		Services:   svcRepo,
		Rules:      availRepo,
		Now:        now,
	}

	bk := &bookings.Application{
		Bookings:  bookRepo,
		Services:  svcRepo,
		Customers: custRepo,
		Payments:  payRepo,
		Events:    notifyOnBookingCreated{n: notifApp},
		Now:       now,
	}

	pay := &payments.Application{
		Payments: payRepo,
		Bookings: bookRepo,
		Services: svcRepo,
		Provider: paymentstub.Provider{},
		Now:      now,
	}

	conv := &conversations.Application{
		Messaging:         msgRepo,
		Customers:         crm,
		Bookings:          bk,
		Tenancy:           ten,
		Outbound:          whatsappOut,
		Now:               now,
		WhatsAppAppSecret: os.Getenv("WHATSAPP_APP_SECRET"),
	}

	tw := &twilio.Application{
		AuthToken: os.Getenv("TWILIO_AUTH_TOKEN"),
	}

	deps := &httpapi.Deps{
		Tenancy:               ten,
		Catalog:               cat,
		Customers:             crm,
		Scheduling:            sch,
		Bookings:              bk,
		Payments:              pay,
		Conversations:         conv,
		Twilio:                tw,
		Notifications:         notifApp,
		PlatformAPIKey:        os.Getenv("PLATFORM_API_KEY"),
		RequirePlatformAPIKey: os.Getenv("REQUIRE_PLATFORM_API_KEY") == "true",
		SkipTenantCheck:       os.Getenv("SKIP_TENANT_CHECK") == "true",
	}

	hard := httpapi.HardeningConfig{
		CORSAllowedOrigins: httpapi.ParseCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS")),
		RateLimitMax:       httpapi.ParsePositiveInt(os.Getenv("HTTP_RATE_LIMIT_MAX"), 0),
		RateLimitWindow:    httpapi.ParseDurationSeconds(os.Getenv("HTTP_RATE_LIMIT_WINDOW_SEC"), time.Minute),
	}
	r := httpapi.NewRouter(httpapi.RouterConfig{
		Stage:     os.Getenv("API_GATEWAY_STAGE"),
		Phase:     phase0.Phase(),
		Hardening: hard,
	}, deps)
	adapter := chiadapter.New(r)
	lambda.Start(adapter.ProxyWithContext)
}
