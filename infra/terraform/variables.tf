variable "aws_region" {
  type        = string
  description = "AWS region"
  default     = "us-east-1"
}

variable "project" {
  type        = string
  description = "Short project name used in resource names"
  default     = "booking"
}

variable "environment" {
  type        = string
  description = "Deployment stage (e.g. dev, staging)"
  default     = "dev"
}

variable "lambda_log_retention_days" {
  type        = number
  description = "CloudWatch log retention for the API Lambda"
  default     = 14
}

variable "dynamodb_point_in_time_recovery" {
  type        = bool
  description = "Force-enable DynamoDB PITR. Also auto-enabled when var.environment is staging/prod-like (see dynamodb.tf locals)."
  default     = false
}

variable "api_gateway_throttle_rate_limit" {
  type        = number
  description = "API Gateway stage steady-state requests per second (REST stage method settings)"
  default     = 500
}

variable "api_gateway_throttle_burst_limit" {
  type        = number
  description = "API Gateway stage burst limit"
  default     = 200
}

variable "alarm_notification_email" {
  type        = string
  description = "If set, subscribe this address to SNS for Lambda alarms (confirm subscription in email)"
  default     = ""
}

variable "monthly_budget_usd" {
  type        = string
  description = "Monthly AWS cost budget limit in USD (used when cost_alert_email is set)"
  default     = "200"
}

variable "cost_alert_email" {
  type        = string
  description = "If set, creates a monthly cost budget with an 80% actual-spend email alert"
  default     = ""
}

variable "budget_start_date" {
  type        = string
  description = "Cost budget period start in YYYY-MM-DD_HH:MM format (UTC)"
  default     = "2026-01-01_00:00"
}
