variable "compartment_ocid" {
  description = "OCID of the compartment that owns the DB system."
  type        = string
}

variable "availability_domain" {
  description = "Tenancy-specific availability-domain name for MySQL."
  type        = string
}

variable "subnet_id" {
  description = "OCID of the private subnet that hosts MySQL."
  type        = string
}

variable "nsg_ids" {
  description = "Network security groups attached to MySQL."
  type        = list(string)
}

variable "name" {
  description = "Display name assigned to the DB system."
  type        = string
}

variable "admin_username" {
  description = "Administrator username created with the DB system."
  type        = string
}

variable "admin_password" {
  description = "Administrator password created with the DB system."
  type        = string
  sensitive   = true
}

variable "database_shape" {
  description = "Always Free shape used by the MySQL DB system."
  type        = string
  default     = "MySQL.Free"

  validation {
    condition     = var.database_shape == "MySQL.Free"
    error_message = "The database shape must remain MySQL.Free."
  }
}

variable "heatwave_shape" {
  description = "Always Free shape used by the HeatWave cluster."
  type        = string
  default     = "HeatWave.Free"

  validation {
    condition     = var.heatwave_shape == "HeatWave.Free"
    error_message = "The HeatWave shape must remain HeatWave.Free."
  }
}
