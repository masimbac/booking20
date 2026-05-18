locals {
  oidc_url         = "https://token.actions.githubusercontent.com"
  github_repo_path = trim(trimspace(var.github_repository), " /")

  oidc_arn = var.create_oidc_provider ? aws_iam_openid_connect_provider.github[0].arn : data.aws_iam_openid_connect_provider.github[0].arn

  staging_subjects = distinct(concat(
    [for ref in var.github_staging_refs : format("repo:%s:ref:%s", local.github_repo_path, trim(trimspace(ref), " /"))],
    var.staging_additional_github_subjects,
    var.trust_staging_github_environment ? [
      format("repo:%s:environment:%s", local.github_repo_path, trim(trimspace(var.staging_github_environment), " /")),
    ] : [],
    var.trust_github_pull_request_workflows ? [
      format("repo:%s:pull_request*", local.github_repo_path),
    ] : [],
  ))

  production_subjects = [
    format("repo:%s:environment:%s", local.github_repo_path, var.production_github_environment),
  ]

  dynamodb_lock_table_arn = "arn:aws:dynamodb:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:table/${var.remote_lock_table_name}"

  booking_table_arn             = "arn:aws:dynamodb:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:table/${var.booking_resource_prefix}-*"
  booking_lambda_arn            = "arn:aws:lambda:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:function:${var.booking_resource_prefix}-*"
  booking_iam_role_arn          = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.booking_resource_prefix}-*"
  booking_log_groups_arn_prefix = "${var.booking_resource_prefix}-"
  booking_log_groups_arns = [
    format(
      "arn:aws:logs:%s:%s:log-group:/aws/lambda/%s*",
      data.aws_region.current.name,
      data.aws_caller_identity.current.account_id,
      local.booking_log_groups_arn_prefix,
    ),
  ]

  state_bucket_arn = format("arn:aws:s3:::%s", var.remote_state_bucket_name)

  # Objects written under booking/ match scripts/render-backend-config.sh state keys (booking/<env>/…).
  state_object_arns = var.protect_state_bucket_objects ? [
    "${local.state_bucket_arn}/booking/*",
  ] : ["${local.state_bucket_arn}/*"]

  list_bucket_arns = [local.state_bucket_arn]

  state_bucket_list_prefix_condition = var.protect_state_bucket_objects ? [
    "booking/",
    "booking/*",
  ] : []
}

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

data "tls_certificate" "github_actions" {
  count = var.create_oidc_provider ? 1 : 0
  url   = local.oidc_url
}
