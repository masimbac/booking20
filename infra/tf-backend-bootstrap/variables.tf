variable "aws_region" {
  type        = string
  description = "Region for backend resources (bucket + DynamoDB)."
  default     = "us-east-1"
}

variable "state_bucket_name" {
  type        = string
  description = "Globally unique DNS-compliant S3 bucket name for Terraform remote state."
}

variable "lock_table_name" {
  type        = string
  description = "DynamoDB table name used for Terraform state locking (partition key LockID)."
}
