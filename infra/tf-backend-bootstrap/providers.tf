provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project   = "booking"
      Component = "terraform-backend"
      ManagedBy = "terraform"
    }
  }
}
