# Production workspace (`production.tfvars` / GitHub Environment "production") — API Gateway stage is **`prod`** (URLs `…/prod/…`).
# Workload naming: **`booking20-prod-<resource>`** (`environment = "prod"` in this file).
#
# Keep production in a dedicated AWS account or OU if your org requires isolation;
# Terraform only distinguishes stacks via backend state + this file.
#
# Deploy (normally via approved pipeline only):
#   terraform -chdir=infra/terraform apply -var-file=environments/production.tfvars

environment = "prod"
project       = "booking20"
aws_region    = "us-east-1"

# Explicit PITR (also enabled automatically for prod-like environment names).
dynamodb_point_in_time_recovery = true

# Longer retention in prod; staging keeps module default unless overridden there.
lambda_log_retention_days = 30
