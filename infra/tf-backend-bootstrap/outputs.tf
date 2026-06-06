output "state_bucket_name" {
  value       = aws_s3_bucket.tf_state.bucket
  description = "Bucket for Terraform remote state files"
}

output "lock_table_name" {
  value       = aws_dynamodb_table.tf_lock.name
  description = "DynamoDB Terraform lock table name"
}

output "backend_config_hint" {
  description = "Values to put in infra/terraform backend .hcl files (see scripts/render-backend-config.sh)"
  value = {
    bucket         = aws_s3_bucket.tf_state.bucket
    dynamodb_table = aws_dynamodb_table.tf_lock.name
    region         = var.aws_region
  }
}
