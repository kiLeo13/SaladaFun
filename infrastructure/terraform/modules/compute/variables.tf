variable "compartment_ocid" { type = string }
variable "availability_domain" { type = string }
variable "subnet_id" { type = string }
variable "nsg_ids" { type = list(string) }
variable "ssh_public_key" {
  type      = string
  sensitive = true
}
variable "name" { type = string }
variable "shape" {
  type    = string
  default = "VM.Standard.E2.1.Micro"
  validation {
    condition     = var.shape == "VM.Standard.E2.1.Micro"
    error_message = "Padinho is constrained to the Always Free VM.Standard.E2.1.Micro shape."
  }
}
variable "boot_volume_size_gb" {
  type    = number
  default = 50
  validation {
    condition     = var.boot_volume_size_gb >= 47 && var.boot_volume_size_gb <= 100
    error_message = "The boot volume must stay between OCI's image minimum and the Always Free storage budget."
  }
}
