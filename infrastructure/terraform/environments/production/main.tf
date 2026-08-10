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
  availability_domain = var.availability_domain
  subnet_id           = module.network.bot_subnet_id
  nsg_ids             = [module.network.bot_nsg_id]
  ssh_public_key      = var.ssh_public_key
  name                = local.name
}

module "mysql" {
  source = "../../modules/mysql"

  compartment_ocid    = var.compartment_ocid
  availability_domain = var.availability_domain
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
    DISCORD_TOKEN=${jsonencode(var.discord_token)}
    DISCORD_APPLICATION_ID=${var.discord_application_id}
    DISCORD_GUILD_ID=${var.discord_guild_id}
    DISCORD_SYNC_COMMANDS=true
    DATABASE_HOST=${module.mysql.private_ip}
    DATABASE_PORT=3306
    DATABASE_USERNAME=${var.mysql_admin_username}
    DATABASE_PASSWORD=${jsonencode(var.mysql_admin_password)}
    DATABASE_NAME=${var.mysql_database_name}
    DATABASE_MAX_OPEN_CONNECTIONS=5
    DATABASE_MAX_IDLE_CONNECTIONS=2
    DATABASE_CONNECTION_MAX_LIFETIME=30m
    MIGRATIONS_PATH=/app/migrations
    LOG_LEVEL=info
  EOT
  registry_credentials = jsonencode({
    username = var.ghcr_username
    token    = var.ghcr_token
  })
}
