variable "aws_region" {
  type        = string
  description = "AWS region deployed into (IAM is global except where ARNs encode region)."
  default     = "us-east-1"
}

variable "github_repository" {
  type        = string
  description = "GitHub repository in OWNER/NAME form used in OIDC `sub`, e.g. parama/booking-2.0."

  validation {
    condition     = length(split("/", trimspace(var.github_repository))) == 2 && !startswith(var.github_repository, "/") && !endswith(var.github_repository, "/")
    error_message = "github_repository must look like OWNER/NAME with a single slash (no leading/trailing slash)."
  }
}

variable "create_oidc_provider" {
  type        = bool
  description = "Whether to install the GitHub OIDC provider (`token.actions.githubusercontent.com`). If false, it must already exist."
  default     = true
}

variable "remote_state_bucket_name" {
  type        = string
  description = "Terraform state bucket (must match infra/tf-backend-bootstrap and .env-local)."

  validation {
    condition     = replace(var.remote_state_bucket_name, "_", "") == var.remote_state_bucket_name && var.remote_state_bucket_name == lower(var.remote_state_bucket_name)
    error_message = "Use lowercase DNS-compliant bucket names — underscores are commonly invalid/rejected."
  }
}

variable "remote_lock_table_name" {
  type        = string
  description = "DynamoDB Terraform lock table created by infra/tf-backend-bootstrap."
}

variable "booking_resource_prefix" {
  type        = string
  description = "Prefix matching application DynamoDB tables, Lambda functions, and IAM roles created by infra/terraform (align with variables.project, default `booking`)."
  default     = "booking"
}

variable "github_staging_refs" {
  type        = list(string)
  description = "Trusted GitHub JWT `sub` branch refs for staging/trunk deploy (`repo:<this>:ref:<entry>`)."
  default     = ["refs/heads/main"]
}

variable "staging_additional_github_subjects" {
  type        = list(string)
  description = "Extra exact `token.actions.githubusercontent.com:sub` values allowed for the staging role (e.g. repo:OWNER/NAME:pull_request)."
  default     = []
}

variable "trust_staging_github_environment" {
  type        = bool
  description = "Jobs that set `jobs.*.environment` to `staging` yield JWT `repo:…:environment:staging`; include that subject when true (recommended)."
  default     = true
}

variable "staging_github_environment" {
  type        = string
  description = "GitHub environment name used in staging-role trust alongside branch refs (`repo:<repo>:environment:<this>`)."
  default     = "staging"
}

variable "trust_github_pull_request_workflows" {
  type        = bool
  description = "Trust `repo:<github_repository>:pull_request*` JWT subjects on the **staging** deploy role so `terraform plan` can run from `pull_request` workflows (recommended for trunk review)."
  default     = true
}

variable "production_github_environment" {
  type        = string
  description = "GitHub Environments value that must appear in JWT `sub` as `repo:…:environment:<this>` for production."
  default     = "production"
}

variable "staging_role_name" {
  type        = string
  description = "IAM role name assumed from GitHub for staging/trunk deploys."
  default     = "booking-github-deploy-staging"
}

variable "production_role_name" {
  type        = string
  description = "IAM role assumed when GitHub environment is `production`."
  default     = "booking-github-deploy-production"
}

variable "protect_state_bucket_objects" {
  type        = bool
  description = "Restrict S3 state objects to staging/production keys rendered by scripts/render-backend-config.sh (`booking/` prefix)."
  default     = true
}
