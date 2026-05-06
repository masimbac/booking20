package main

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestIsHealth(t *testing.T) {
	tests := []struct {
		name  string
		req   events.APIGatewayProxyRequest
		want  bool
	}{
		{
			name: "exact path",
			req: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/v1/health",
			},
			want: true,
		},
		{
			name: "stage prefixed",
			req: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/dev/v1/health",
			},
			want: true,
		},
		{
			name: "resource match",
			req: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/x",
				Resource:   "/v1/health",
			},
			want: true,
		},
		{
			name: "wrong method",
			req: events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/v1/health",
			},
			want: false,
		},
		{
			name: "other path",
			req: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/v1/bookings",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHealth(tt.req); got != tt.want {
				t.Fatalf("isHealth() = %v, want %v", got, tt.want)
			}
		})
	}
}
