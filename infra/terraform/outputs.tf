output "dynamodb_table_name" {
  description = "DynamoDB single-table name"
  value       = aws_dynamodb_table.core.name
}

output "dynamodb_table_arn" {
  description = "DynamoDB table ARN"
  value       = aws_dynamodb_table.core.arn
}

output "lambda_function_name" {
  description = "API Lambda function name"
  value       = aws_lambda_function.api.function_name
}

output "api_gateway_rest_api_id" {
  description = "REST API id"
  value       = aws_api_gateway_rest_api.this.id
}

output "health_url" {
  description = "Invoke GET /v1/health on the deployed stage"
  value       = "https://${aws_api_gateway_rest_api.this.id}.execute-api.${var.aws_region}.amazonaws.com/${aws_api_gateway_stage.this.stage_name}/v1/health"
}
