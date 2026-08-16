variable "compartment_ocid" {
  description = "OCID of the compartment that owns network resources."
  type        = string
}

variable "name" {
  description = "Display-name prefix for network resources."
  type        = string
}

variable "vcn_cidr" {
  description = "IPv4 CIDR assigned to the Salada VCN."
  type        = string
}

variable "bot_subnet_cidr" {
  description = "IPv4 CIDR assigned to Padinho's public subnet."
  type        = string
}

variable "database_subnet_cidr" {
  description = "IPv4 CIDR assigned to MySQL's private subnet."
  type        = string
}

variable "admin_cidr" {
  description = "Administrator's public IPv4 address in /32 notation."
  type        = string

  validation {
    condition = (
      can(cidrhost(var.admin_cidr, 0)) &&
      endswith(var.admin_cidr, "/32")
    )
    error_message = "admin_cidr must be the administrator's single IPv4 address in /32 notation."
  }
}
