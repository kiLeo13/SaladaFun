variable "tenancy_ocid" {
  description = "OCID of the tenancy used to resolve availability domains."
  type        = string
}

variable "compartment_ocid" {
  description = "OCID of the compartment that owns Salada resources."
  type        = string
}

variable "region" {
  description = "OCI region where Salada is deployed."
  type        = string
}

variable "availability_domain_number" {
  description = "OCI availability-domain number used by Compute and MySQL."
  type        = number
  default     = 3

  validation {
    condition     = var.availability_domain_number >= 1 && floor(var.availability_domain_number) == var.availability_domain_number
    error_message = "availability_domain_number must be a positive integer."
  }
}

variable "admin_cidr" {
  description = "Administrator's public IPv4 address in /32 notation."
  type        = string
}

variable "ssh_public_key" {
  description = "OpenSSH public key authorized on the Padinho VM."
  type        = string
  sensitive   = true
}

variable "mysql_admin_username" {
  description = "Administrator username created with the MySQL DB system."
  type        = string
  default     = "salada_admin"
}

variable "mysql_admin_password" {
  description = "Administrator password created with the MySQL DB system."
  type        = string
  sensitive   = true

  validation {
    condition = (
      length(var.mysql_admin_password) >= 8 &&
      length(var.mysql_admin_password) <= 32 &&
      can(regex("^[ -~]+$", var.mysql_admin_password)) &&
      !strcontains(var.mysql_admin_password, "$") &&
      !strcontains(var.mysql_admin_password, "`") &&
      !strcontains(var.mysql_admin_password, "\\") &&
      !strcontains(var.mysql_admin_password, "\"")
    )
    error_message = "mysql_admin_password must be 8-32 printable characters without $, backticks, backslashes, or double quotes."
  }
}

variable "mysql_database_name" {
  description = "MySQL schema consumed by migrations and Padinho."
  type        = string
  default     = "salada"

  validation {
    condition     = can(regex("^[a-z][a-z0-9_]*$", var.mysql_database_name))
    error_message = "mysql_database_name must be a lowercase MySQL identifier."
  }
}

variable "mysql_hostname_label" {
  description = "DNS hostname label assigned to the MySQL DB system."
  type        = string
  default     = "saladadb"
}

variable "vcn_dns_label" {
  description = "DNS label assigned to the Salada VCN."
  type        = string
  default     = "saladafun"
}
