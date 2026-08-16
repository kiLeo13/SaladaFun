terraform {
  required_version = ">= 1.15.5"

  required_providers {
    oci = {
      source  = "oracle/oci"
      version = ">= 8.20.0, < 9.0.0"
    }

    random = {
      source  = "hashicorp/random"
      version = "~> 3.9"
    }
  }
}

provider "oci" {
  region = var.region
}
