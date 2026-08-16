locals {
  name = "salada"
}

module "network" {
  source = "../../modules/network"

  compartment_ocid     = var.compartment_ocid
  name                 = local.name
  vcn_cidr             = "10.42.0.0/16"
  bot_subnet_cidr      = "10.42.10.0/24"
  database_subnet_cidr = "10.42.20.0/24"
  admin_cidr           = var.admin_cidr
}

module "compute" {
  source = "../../modules/compute"

  compartment_ocid    = var.compartment_ocid
  availability_domain = data.oci_identity_availability_domain.selected.name
  subnet_id           = module.network.bot_subnet_id
  nsg_ids             = [module.network.bot_nsg_id]
  ssh_public_key      = var.ssh_public_key
  name                = local.name
}

module "mysql" {
  source = "../../modules/mysql"

  compartment_ocid    = var.compartment_ocid
  availability_domain = data.oci_identity_availability_domain.selected.name
  subnet_id           = module.network.database_subnet_id
  nsg_ids             = [module.network.database_nsg_id]
  name                = local.name
  admin_username      = var.mysql_admin_username
  admin_password      = var.mysql_admin_password
}

module "identity" {
  source = "../../modules/identity"

  tenancy_ocid        = var.tenancy_ocid
  compartment_ocid    = var.compartment_ocid
  instance_id         = module.compute.instance_id
  name                = local.name
  runtime_environment = <<-EOT
    DB_HOST=${module.mysql.private_ip}
    DB_PORT=3306
    DB_USER=${var.mysql_admin_username}
    DB_PASSWORD=${jsonencode(var.mysql_admin_password)}
    DB_NAME=${var.mysql_database_name}
    DB_MAX_OPEN=5
    DB_MAX_IDLE=2
    DB_MAX_LIFETIME=30m
  EOT
}
