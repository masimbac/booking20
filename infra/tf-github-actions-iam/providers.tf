provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "booking"
      Component   = "github-actions-terraform"
      ManagedBy   = "terraform"
      Environment = "shared"
    }
  }
}

provider "tls" {}
