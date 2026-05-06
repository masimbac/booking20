.PHONY: test lint fmt fmt-check vet build-lambda clean \
	terraform-init terraform-fmt terraform-validate terraform-plan terraform-apply

GO ?= go
LAMBDA_ARTIFACT ?= bin/bootstrap
TF_DIR := infra/terraform

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

terraform-init:
	terraform -chdir=$(TF_DIR) init

terraform-fmt:
	terraform -chdir=$(TF_DIR) fmt -recursive

terraform-validate: build-lambda
	terraform -chdir=$(TF_DIR) init -backend=false -input=false
	terraform -chdir=$(TF_DIR) validate

terraform-plan: build-lambda
	terraform -chdir=$(TF_DIR) plan

terraform-apply: build-lambda
	terraform -chdir=$(TF_DIR) apply

clean:
	rm -rf bin $(TF_DIR)/.build
