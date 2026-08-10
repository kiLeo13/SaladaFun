# Terraform

Terraform provisions only resources selected for OCI Always Free: one
`VM.Standard.E2.1.Micro`, a `MySQL.Free` DB system with one `HeatWave.Free`
node, Object Storage state, Vault secrets, and the required networking. The VM
has a public IP only because Always Free has no NAT Gateway allowance. Its NSG
admits TCP/22 solely from `admin_cidr`; no Padinho application port is exposed.
MySQL has no public IP and accepts TCP/3306 only from the bot NSG.

## Bootstrap state storage

The committed `backend.tf` files intentionally contain an empty OCI backend.
Copy both `.example` configuration files without the `.example` suffix; the
real `backend.hcl` and `terraform.tfvars` files are ignored.

Create the bucket once using local state, then migrate that bootstrap state into
the bucket created by the same configuration:

```sh
cd infrastructure/terraform/bootstrap
terraform init -backend=false
terraform apply
cp backend.hcl.example backend.hcl
# Replace example values with the apply outputs.
terraform init -migrate-state -backend-config=backend.hcl
```

The native OCI backend requires Terraform 1.12 or newer. It stores state at the
configured key and automatically creates `<key>.lock` in the same versioned,
private bucket during state-changing operations.

The similarly named files serve different purposes: `terraform.tfvars` is the
real, automatically loaded input file and stays ignored;
`terraform.tfvars.example` is its committed template. `.terraform.lock.hcl` is
the committed provider dependency lock. The transient state lock is the
`<key>.lock` object in OCI Object Storage, not a local file checked into Git.

## Production

```sh
cd infrastructure/terraform/environments/production
cp backend.hcl.example backend.hcl
cp terraform.tfvars.example terraform.tfvars
terraform init -backend-config=backend.hcl
terraform plan
terraform apply
```

Never commit either copied file. Treat Terraform state as sensitive because the
Vault secret payload necessarily passes through state. OCI credentials should
come from the standard OCI config/environment rather than HCL.
