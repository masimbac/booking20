package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"

	"github.com/parama/booking/internal/adapters/dynamo"
	"github.com/parama/booking/internal/adapters/outbound"
	"github.com/parama/booking/internal/app/bookings"
	"github.com/parama/booking/internal/app/catalog"
	"github.com/parama/booking/internal/app/conversations"
	"github.com/parama/booking/internal/app/customers"
	"github.com/parama/booking/internal/app/scheduling"
	"github.com/parama/booking/internal/app/tenancy"
	"github.com/parama/booking/internal/httpapi"
	"github.com/parama/booking/internal/phase0"
)

func main() {
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

	now := func() time.Time { return time.Now().UTC() }
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
		Now:       now,
	}

	conv := &conversations.Application{
		Messaging:         msgRepo,
		Customers:         crm,
		Bookings:          bk,
		Tenancy:           ten,
		Outbound:          outbound.WhatsAppStub{},
		Now:               now,
		WhatsAppAppSecret: os.Getenv("WHATSAPP_APP_SECRET"),
	}

	deps := &httpapi.Deps{
		Tenancy:         ten,
		Catalog:         cat,
		Customers:       crm,
		Scheduling:      sch,
		Bookings:        bk,
		Conversations:   conv,
		PlatformAPIKey:  os.Getenv("PLATFORM_API_KEY"),
		SkipTenantCheck: os.Getenv("SKIP_TENANT_CHECK") == "true",
	}

	r := httpapi.NewRouter(httpapi.RouterConfig{
		Stage: os.Getenv("API_GATEWAY_STAGE"),
		Phase: phase0.Phase(),
	}, deps)
	adapter := chiadapter.New(r)
	lambda.Start(adapter.ProxyWithContext)
}
