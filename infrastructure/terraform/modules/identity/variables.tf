variable "tenancy_ocid" {
  description = "OCID of the OCI tenancy that owns the dynamic group."
  type        = string
}

variable "compartment_ocid" {
  description = "OCID of the compartment that owns Vault and IAM resources."
  type        = string
}

variable "instance_id" {
  description = "OCID of the instance granted access to runtime secrets."
  type        = string
}

variable "name" {
  description = "Display-name prefix for identity and Vault resources."
  type        = string
}

variable "runtime_environment" {
  description = "Dotenv payload stored as Padinho's runtime secret."
  type        = string
  sensitive   = true
}
