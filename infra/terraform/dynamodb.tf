locals {
  name_prefix = "${var.project}-${var.environment}"
  # PITR for DynamoDB: explicit var, or automatically on staging/prod-style environments.
  dynamodb_pitr_enabled = var.dynamodb_point_in_time_recovery || contains(
    ["staging", "stg", "prod", "production", "prd"], var.environment,
  )
}

resource "aws_dynamodb_table" "core" {
  name         = "${local.name_prefix}-core"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "PK"
  range_key    = "SK"

  attribute {
    name = "PK"
    type = "S"
  }
  attribute {
    name = "SK"
    type = "S"
  }
  attribute {
    name = "GSI1PK"
    type = "S"
  }
  attribute {
    name = "GSI1SK"
    type = "S"
  }
  attribute {
    name = "GSI2PK"
    type = "S"
  }
  attribute {
    name = "GSI2SK"
    type = "S"
  }
  attribute {
    name = "GSI3PK"
    type = "S"
  }
  attribute {
    name = "GSI3SK"
    type = "S"
  }
  attribute {
    name = "GSI4PK"
    type = "S"
  }
  attribute {
    name = "GSI4SK"
    type = "S"
  }

  # Bookings by date — see DYNAMODB_ARCHITECTURE_AND_SCHEMA.md (GSI1)
  global_secondary_index {
    name            = "GSI1"
    hash_key        = "GSI1PK"
    range_key       = "GSI1SK"
    projection_type = "ALL"
  }

  # Customer by phone + composite SK for tenant scoping (GSI2)
  global_secondary_index {
    name            = "GSI2"
    hash_key        = "GSI2PK"
    range_key       = "GSI2SK"
    projection_type = "ALL"
  }

  # Payments by booking (GSI3)
  global_secondary_index {
    name            = "GSI3"
    hash_key        = "GSI3PK"
    range_key       = "GSI3SK"
    projection_type = "ALL"
  }

  # Notifications by schedule (GSI4)
  global_secondary_index {
    name            = "GSI4"
    hash_key        = "GSI4PK"
    range_key       = "GSI4SK"
    projection_type = "ALL"
  }

  point_in_time_recovery {
    enabled = local.dynamodb_pitr_enabled
  }

  server_side_encryption {
    enabled = true
  }
}
