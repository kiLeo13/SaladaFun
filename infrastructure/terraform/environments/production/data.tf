data "oci_identity_availability_domain" "selected" {
  compartment_id = var.tenancy_ocid
  ad_number      = var.availability_domain_number
}
