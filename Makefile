.PHONY: test lint fmt fmt-check vet build-lambda clean

GO ?= go
LAMBDA_ARTIFACT ?= bin/bootstrap

test:
	$(GO) test -count=1 -race ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

fmt-check:
	@test -z "$$($(GO) fmt ./... | tee /dev/stderr)"

lint:
	golangci-lint run

# Lambda runtime on AWS Graviton uses arm64; override: make build-lambda LAMBDA_GOARCH=amd64
LAMBDA_GOARCH ?= arm64

build-lambda:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=$(LAMBDA_GOARCH) $(GO) build -trimpath -ldflags="-s -w" -o $(LAMBDA_ARTIFACT) ./cmd/api

clean:
	rm -rf bin
