resource "aws_iam_openid_connect_provider" "github" {
  count = var.create_oidc_provider ? 1 : 0

  url = local.oidc_url

  client_id_list = [
    "sts.amazonaws.com",
  ]

  thumbprint_list = distinct([for c in data.tls_certificate.github_actions[0].certificates : c.sha1_fingerprint])
}

data "aws_iam_openid_connect_provider" "github" {
  count = var.create_oidc_provider ? 0 : 1
  url   = local.oidc_url
}
