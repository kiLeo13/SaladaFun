variable "compartment_ocid" {
  description = "OCID of the compartment that owns the instance."
  type        = string
}

variable "availability_domain" {
  description = "Tenancy-specific availability-domain name for the instance."
  type        = string
}

variable "subnet_id" {
  description = "OCID of the subnet attached to the primary VNIC."
  type        = string
}

variable "nsg_ids" {
  description = "Network security groups attached to the primary VNIC."
  type        = list(string)
}

variable "ssh_public_key" {
  description = "OpenSSH public key authorized on the instance."
  type        = string
  sensitive   = true
}

variable "name" {
  description = "Display-name prefix for compute resources."
  type        = string
}

variable "shape" {
  description = "Always Free compute shape used by Padinho."
  type        = string
  default     = "VM.Standard.E2.1.Micro"

  validation {
    condition     = var.shape == "VM.Standard.E2.1.Micro"
    error_message = "Salada is constrained to the Always Free VM.Standard.E2.1.Micro shape."
  }
}

variable "boot_volume_size_gb" {
  description = "Boot volume size in gigabytes."
  type        = number
  default     = 50

  validation {
    condition = (
      var.boot_volume_size_gb >= 47 &&
      var.boot_volume_size_gb <= 100
    )
    error_message = "The boot volume must stay between OCI's image minimum and the Always Free storage budget."
  }
}
