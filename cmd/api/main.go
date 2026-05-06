package main

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/parama/booking/internal/phase0"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	_ = ctx // reserved for future timeouts / tracing
	if isHealth(req) {
		body, err := json.Marshal(map[string]any{
			"status": "ok",
			"phase":  phase0.Phase(),
		})
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: string(body),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 404,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"title":"Not Found","status":404}`,
	}, nil
}

func isHealth(req events.APIGatewayProxyRequest) bool {
	if req.HTTPMethod != "GET" {
		return false
	}
	switch {
	case req.Path == "/v1/health":
		return true
	case req.Resource == "/v1/health":
		return true
	case strings.HasSuffix(req.Path, "/v1/health"):
		// REST API stage-prefixed paths, e.g. /dev/v1/health
		return true
	default:
		return false
	}
}
