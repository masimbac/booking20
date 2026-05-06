data "archive_file" "api" {
  type             = "zip"
  source_file      = abspath("${path.module}/../../bin/bootstrap")
  output_path      = "${path.module}/.build/api-lambda.zip"
  output_file_mode = "0644"
}

resource "aws_cloudwatch_log_group" "api_lambda" {
  name              = "/aws/lambda/${local.name_prefix}-api"
  retention_in_days = var.lambda_log_retention_days
}

resource "aws_lambda_function" "api" {
  function_name = "${local.name_prefix}-api"
  role          = aws_iam_role.api_lambda.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]

  filename         = data.archive_file.api.output_path
  source_code_hash = data.archive_file.api.output_base64sha256

  memory_size = 128
  timeout     = 10

  depends_on = [
    aws_cloudwatch_log_group.api_lambda,
  ]
}
