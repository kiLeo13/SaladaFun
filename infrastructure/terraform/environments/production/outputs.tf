output "bot_public_ip" {
  description = "Public IP used only for SSH from admin_cidr."
  value       = module.compute.public_ip
}

output "runtime_secret_ocid" {
  description = "Vault secret fetched by Ansible through the instance principal."
  value       = module.identity.runtime_secret_ocid
}

output "registry_secret_ocid" {
  description = "Vault secret containing the read-only GHCR credentials."
  value       = module.identity.registry_secret_ocid
}

output "mysql_private_ip" {
  value = module.mysql.private_ip
}
