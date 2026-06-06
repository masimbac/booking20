data "aws_iam_policy_document" "github_terraform_deploy" {
  statement {
    sid    = "RemoteStateRW"
    effect = "Allow"
    actions = [
      "s3:GetObject",
      "s3:GetObjectTagging",
      "s3:GetObjectVersion",
      "s3:GetObjectVersionTagging",
      "s3:PutObject",
      "s3:PutObjectTagging",
      "s3:DeleteObject",
      "s3:ListBucketMultipartUploads",
      "s3:AbortMultipartUpload",
    ]
    resources = local.state_object_arns
  }

  statement {
    sid    = "RemoteStateBucketList"
    effect = "Allow"
    actions = [
      "s3:ListBucket",
      "s3:GetBucketLocation",
      "s3:GetBucketEncryption",
      "s3:GetBucketPublicAccessBlock",
      "s3:GetBucketPolicy",
      "s3:GetBucketVersioning",
    ]
    resources = local.list_bucket_arns

    dynamic "condition" {
      for_each = var.protect_state_bucket_objects ? [1] : []
      content {
        test     = "StringLike"
        variable = "s3:prefix"
        values   = local.state_bucket_list_prefix_condition
      }
    }
  }

  statement {
    sid    = "RemoteStateLocks"
    effect = "Allow"
    actions = [
      "dynamodb:DescribeTable",
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:DeleteItem",
      "dynamodb:ConditionCheckItem",
    ]
    resources = [local.dynamodb_lock_table_arn]
  }

  statement {
    sid    = "CoreDynamoRW"
    effect = "Allow"
    actions = [
      "dynamodb:*",
    ]
    resources = [
      local.booking_table_arn,
      "${local.booking_table_arn}/index/*",
    ]
  }

  statement {
    sid    = "LambdaWorkload"
    effect = "Allow"
    actions = [
      "lambda:*",
    ]
    resources = [
      local.booking_lambda_arn,
    ]
  }

  statement {
    sid    = "WorkloadLogs"
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:DescribeLogGroups",
      "logs:DeleteLogGroup",
      "logs:PutRetentionPolicy",
      "logs:TagLogGroup",
      "logs:ListTagsLogGroup",
      "logs:ListTagsForResource",
    ]
    resources = local.booking_log_groups_arns
  }

  statement {
    sid    = "WorkloadLogsDescribe"
    effect = "Allow"
    actions = [
      "logs:DescribeLogStreams",
      "logs:DeleteLogStream",
      "logs:FilterLogEvents",
    ]
    resources = ["*"]
  }

  statement {
    sid    = "ApigwRegional"
    effect = "Allow"
    actions = [
      "apigateway:GET",
      "apigateway:POST",
      "apigateway:PUT",
      "apigateway:PATCH",
      "apigateway:DELETE",
    ]
    resources = [
      "arn:aws:apigateway:${data.aws_region.current.name}::/restapis/*",
      "arn:aws:apigateway:${data.aws_region.current.name}::/account",
    ]
  }

  statement {
    sid       = "ApigwTaggingShared"
    effect    = "Allow"
    actions   = ["tag:GetResources"]
    resources = ["*"]
  }

  statement {
    sid    = "IamServiceRolesBooking"
    effect = "Allow"
    actions = [
      "iam:CreateRole",
      "iam:DeleteRole",
      "iam:GetRole",
      "iam:TagRole",
      "iam:UntagRole",
      "iam:UpdateRoleDescription",
      "iam:PutRolePolicy",
      "iam:DeleteRolePolicy",
      "iam:GetRolePolicy",
      "iam:ListAttachedRolePolicies",
      "iam:ListRolePolicies",
      "iam:ListRoleTags",
    ]
    resources = [
      local.booking_iam_role_arn,
    ]
  }

  statement {
    sid    = "IamAwsManagedLambdaBasicRead"
    effect = "Allow"
    actions = [
      "iam:GetPolicy",
      "iam:GetPolicyVersion",
      "iam:ListPolicyVersions",
    ]
    resources = [
      "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
    ]
  }

  statement {
    sid    = "IamGithubDeployTerraformPolicyRW"
    effect = "Allow"
    actions = [
      "iam:GetPolicy",
      "iam:GetPolicyVersion",
      "iam:ListPolicyVersions",
      "iam:CreatePolicyVersion",
      "iam:DeletePolicyVersion",
      "iam:SetDefaultPolicyVersion",
      "iam:TagPolicy",
      "iam:UntagPolicy",
      "iam:ListPolicyTags",
    ]
    resources = [
      "arn:aws:iam::${data.aws_caller_identity.current.account_id}:policy/${var.booking_resource_prefix}-github-*",
    ]
  }

  statement {
    sid       = "PassRoleBookingLambdaExecution"
    effect    = "Allow"
    actions   = ["iam:PassRole"]
    resources = [local.booking_iam_role_arn]

    condition {
      test     = "StringEquals"
      variable = "iam:PassedToService"
      values   = ["lambda.amazonaws.com"]
    }
  }

  statement {
    sid       = "PolicyAttachmentAwsManagedLambdaBasic"
    effect    = "Allow"
    actions   = ["iam:AttachRolePolicy", "iam:DetachRolePolicy"]
    resources = [local.booking_iam_role_arn]

    condition {
      test     = "ArnLike"
      variable = "iam:PolicyArn"
      values   = ["arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"]
    }
  }

  statement {
    sid    = "CloudWatchAlarms"
    effect = "Allow"
    actions = [
      "cloudwatch:PutMetricAlarm",
      "cloudwatch:DeleteAlarms",
      "cloudwatch:DescribeAlarms",
      "cloudwatch:SetAlarmState",
      "cloudwatch:TagResource",
      "cloudwatch:UntagResource",
      "cloudwatch:ListTagsForResource",
    ]
    resources = [
      format("arn:aws:cloudwatch:%s:%s:alarm:%s-*-*", data.aws_region.current.name, data.aws_caller_identity.current.account_id, var.booking_resource_prefix),
    ]
  }

  statement {
    sid    = "SNSOptional"
    effect = "Allow"
    actions = [
      "sns:CreateTopic",
      "sns:GetTopicAttributes",
      "sns:SetTopicAttributes",
      "sns:DeleteTopic",
      "sns:Subscribe",
      "sns:Unsubscribe",
      "sns:GetSubscriptionAttributes",
      "sns:SetSubscriptionAttributes",
      "sns:ListSubscriptionsByTopic",
      "sns:AddPermission",
      "sns:RemovePermission",
      "sns:TagResource",
      "sns:UntagResource",
      "sns:ListTagsForResource",
    ]
    resources = [
      "arn:aws:sns:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:${var.booking_resource_prefix}-*",
    ]
  }

  statement {
    sid    = "BudgetsWorkload"
    effect = "Allow"
    actions = [
      "budgets:*",
    ]
    resources = ["*"]
  }

  statement {
    sid       = "StsCallerIdentityRead"
    effect    = "Allow"
    actions   = ["sts:GetCallerIdentity"]
    resources = ["*"]
  }

}

resource "aws_iam_policy" "github_terraform_deploy" {
  name   = substr("${var.booking_resource_prefix}-github-terraform-deploy", 0, 127)
  path   = "/"
  policy = data.aws_iam_policy_document.github_terraform_deploy.json
}

resource "aws_iam_role_policy_attachment" "staging_terraform_deploy" {
  role       = aws_iam_role.staging.name
  policy_arn = aws_iam_policy.github_terraform_deploy.arn
}

resource "aws_iam_role_policy_attachment" "production_terraform_deploy" {
  role       = aws_iam_role.production.name
  policy_arn = aws_iam_policy.github_terraform_deploy.arn
}
