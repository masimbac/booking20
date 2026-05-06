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
  description = "Enable DynamoDB PITR (extra cost)"
  default     = false
}
