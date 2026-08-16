data "oci_objectstorage_namespace" "current" {
  compartment_id = var.compartment_ocid
}

resource "random_id" "state_bucket" {
  byte_length = 16
}

resource "oci_objectstorage_bucket" "terraform_state" {
  compartment_id = var.compartment_ocid
  namespace      = data.oci_objectstorage_namespace.current.namespace
  name           = random_id.state_bucket.hex
  access_type    = "NoPublicAccess"
  storage_tier   = "Standard"
  auto_tiering   = "Disabled"

  lifecycle {
    prevent_destroy = true
  }
}
