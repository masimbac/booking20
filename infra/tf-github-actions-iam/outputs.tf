output "oidc_provider_arn" {
  description = "Federated IAM OIDC ARN used in assume-role policies."
  value       = local.oidc_arn
}

output "staging_deploy_role_arn" {
  description = "ARN for GitHub workflows that deploy infra from main/trunk (staging stack)."
  value       = aws_iam_role.staging.arn
}

output "production_deploy_role_arn" {
  description = "ARN for manual production workflows guarded by GitHub Environment `production`."
  value       = aws_iam_role.production.arn
}

output "staging_trusted_subjects" {
  description = "GitHub JWT `sub` values accepted for staging role."
  value       = local.staging_subjects
}

output "production_trusted_subjects" {
  description = "GitHub JWT `sub` values accepted for production role."
  value       = local.production_subjects
}
