data "oci_core_images" "ubuntu" {
  compartment_id           = var.compartment_ocid
  operating_system         = "Canonical Ubuntu"
  operating_system_version = "24.04"
  shape                    = var.shape
  sort_by                  = "TIMECREATED"
  sort_order               = "DESC"
}

resource "oci_core_instance" "bot" {
  availability_domain = var.availability_domain
  compartment_id      = var.compartment_ocid
  display_name        = var.name
  shape               = var.shape

  create_vnic_details {
    assign_public_ip = true
    display_name     = "${var.name}-primary"
    hostname_label   = "padinho"
    nsg_ids          = var.nsg_ids
    subnet_id        = var.subnet_id
  }

  metadata = {
    ssh_authorized_keys = var.ssh_public_key
  }

  source_details {
    source_id               = data.oci_core_images.ubuntu.images[0].id
    source_type             = "image"
    boot_volume_size_in_gbs = var.boot_volume_size_gb
  }

  lifecycle {
    precondition {
      condition     = length(data.oci_core_images.ubuntu.images) > 0
      error_message = "No Ubuntu 24.04 image supports VM.Standard.E2.1.Micro in this region."
    }
  }
}
