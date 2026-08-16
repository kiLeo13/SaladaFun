output "bot_public_ip" {
  description = "Public IP used only for SSH from admin_cidr."
  value       = module.compute.public_ip
}

output "runtime_secret_ocid" {
  description = "Vault secret fetched by Ansible through the instance principal."
  value       = module.identity.runtime_secret_ocid
}

output "mysql_private_ip" {
  description = "Private IPv4 address reached only from the Padinho subnet."
  value       = module.mysql.private_ip
}

output "availability_domain" {
  description = "Tenancy-specific availability-domain name selected by number."
  value       = data.oci_identity_availability_domain.selected.name
}
