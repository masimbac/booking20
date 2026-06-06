.PHONY: test lint fmt fmt-check vet build-lambda clean \
	terraform-backend-config terraform-backend-bootstrap-init terraform-backend-bootstrap-apply \
	terraform-init terraform-init-staging terraform-init-production \
	terraform-github-actions-iam-init terraform-github-actions-iam-validate \
	terraform-github-actions-iam-plan terraform-github-actions-iam-apply \
	install-gh github-actions-ensure-environments github-actions-sync github-actions-repo-setup \
	terraform-fmt terraform-validate terraform-plan terraform-apply

GO ?= go
GITHUB_REPO :=
LAMBDA_ARTIFACT ?= bin/bootstrap
TF_DIR := infra/terraform
TFGHA_DIR := infra/tf-github-actions-iam
TF_BACKEND_STAGING := $(CURDIR)/$(TF_DIR)/.backend/staging.hcl
TF_BACKEND_PROD := $(CURDIR)/$(TF_DIR)/.backend/production.hcl

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

terraform-backend-bootstrap-init:
	terraform -chdir=infra/tf-backend-bootstrap init

terraform-backend-bootstrap-apply:
	terraform -chdir=infra/tf-backend-bootstrap apply

terraform-github-actions-iam-init:
	terraform -chdir=$(TFGHA_DIR) init -backend=false

terraform-github-actions-iam-validate:
	terraform -chdir=$(TFGHA_DIR) fmt -check -recursive
	terraform -chdir=$(TFGHA_DIR) init -backend=false -input=false
	terraform -chdir=$(TFGHA_DIR) validate

terraform-github-actions-iam-plan:
	terraform -chdir=$(TFGHA_DIR) plan

terraform-github-actions-iam-apply:
	terraform -chdir=$(TFGHA_DIR) apply

install-gh:
	@if command -v gh >/dev/null 2>&1; then \
	  $$(gh --version | head -n1); \
	elif command -v brew >/dev/null 2>&1; then \
	  brew install gh; \
	else \
	  printf '%s\n' 'Install the GitHub CLI from https://cli.github.com/installation' >&2; exit 1; fi

# Requires: gh auth login. Override remote: make GITHUB_REPO=owner/repo github-actions-sync
GITHUB_SYNC_FLAGS := $(if $(strip $(GITHUB_REPO)),-R $(GITHUB_REPO),)

github-actions-ensure-environments:
	chmod +x scripts/sync-github-actions-repo-config.sh
	./scripts/sync-github-actions-repo-config.sh $(GITHUB_SYNC_FLAGS) --ensure-environments

github-actions-sync:
	chmod +x scripts/sync-github-actions-repo-config.sh
	./scripts/sync-github-actions-repo-config.sh $(GITHUB_SYNC_FLAGS)

github-actions-repo-setup: install-gh github-actions-ensure-environments github-actions-sync

terraform-backend-config:
	chmod +x scripts/render-backend-config.sh
	./scripts/render-backend-config.sh

# Remote state backend (S3): requires repo-root `.env-local` matching `.env-local.example`.
terraform-init: terraform-backend-config
	terraform -chdir=$(TF_DIR) init -backend-config=$(TF_BACKEND_STAGING)

terraform-init-staging: terraform-backend-config
	terraform -chdir=$(TF_DIR) init -backend-config=$(TF_BACKEND_STAGING)

terraform-init-production: terraform-backend-config
	terraform -chdir=$(TF_DIR) init -backend-config=$(TF_BACKEND_PROD)

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
