data "aws_iam_policy_document" "assume_github_staging" {
  statement {
    sid     = "GitHubOIDCStaging"
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [local.oidc_arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = local.staging_subjects
    }
  }
}

data "aws_iam_policy_document" "assume_github_production" {
  statement {
    sid     = "GitHubOIDCProduction"
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [local.oidc_arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = local.production_subjects
    }
  }
}

resource "aws_iam_role" "staging" {
  name               = var.staging_role_name
  assume_role_policy = data.aws_iam_policy_document.assume_github_staging.json
}

resource "aws_iam_role" "production" {
  name               = var.production_role_name
  assume_role_policy = data.aws_iam_policy_document.assume_github_production.json
}
