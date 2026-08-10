resource "oci_kms_vault" "this" {
  compartment_id = var.compartment_ocid
  display_name   = "${var.name}-vault"
  vault_type     = "DEFAULT"
}

resource "oci_kms_key" "this" {
  compartment_id      = var.compartment_ocid
  display_name        = "${var.name}-secrets"
  management_endpoint = oci_kms_vault.this.management_endpoint
  key_shape {
    algorithm = "AES"
    length    = 32
  }
}

resource "oci_vault_secret" "runtime_environment" {
  compartment_id = var.compartment_ocid
  key_id         = oci_kms_key.this.id
  secret_name    = "${var.name}-runtime-environment"
  vault_id       = oci_kms_vault.this.id
  secret_content {
    content_type = "BASE64"
    content      = base64encode(var.runtime_environment)
  }
}

resource "oci_vault_secret" "registry_credentials" {
  compartment_id = var.compartment_ocid
  key_id          = oci_kms_key.this.id
  secret_name     = "${var.name}-registry-credentials"
  vault_id        = oci_kms_vault.this.id
  secret_content {
    content_type = "BASE64"
    content      = base64encode(var.registry_credentials)
  }
}

resource "oci_identity_dynamic_group" "bot" {
  compartment_id = var.tenancy_ocid
  name           = "${replace(var.name, "-", "_")}_instances"
  description    = "Instances allowed to read Salada runtime secrets"
  matching_rule  = "ALL {instance.id = '${var.instance_id}'}"
}

resource "oci_identity_policy" "bot_secrets" {
  compartment_id = var.compartment_ocid
  name           = "${replace(var.name, "-", "_")}_read_secrets"
  description    = "Allow Salada instances to read only secret bundles"
  statements = [
    "Allow dynamic-group ${oci_identity_dynamic_group.bot.name} to read secret-bundles in compartment id ${var.compartment_ocid}"
  ]
}
