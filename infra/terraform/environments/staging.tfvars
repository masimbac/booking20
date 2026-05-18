# Staging workspace — naming: booking-staging-<resource> (project + environment).
#
# Deploy (after remote state + credentials are wired):
#   terraform -chdir=infra/terraform apply -var-file=environments/staging.tfvars

environment = "staging"
project     = "booking"
aws_region  = "us-east-1"

# PITR is auto-enabled when environment=staging (see dynamodb.tf). Override if needed.
