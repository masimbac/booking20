package main

import (
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"

	"github.com/parama/booking/internal/httpapi"
	"github.com/parama/booking/internal/phase0"
)

func main() {
	r := httpapi.NewRouter(httpapi.RouterConfig{
		Stage: os.Getenv("API_GATEWAY_STAGE"),
		Phase: phase0.Phase(),
	})
	adapter := chiadapter.New(r)
	lambda.Start(adapter.ProxyWithContext)
}
