# Terraform

Terraform provisions only resources selected for OCI Always Free: one
`VM.Standard.E2.1.Micro`, a `MySQL.Free` DB system with one `HeatWave.Free`
node, Object Storage state, and the required networking. The VM has a public IP
only because Always Free has no NAT Gateway allowance. Its NSG admits TCP/22
solely from `admin_cidr`; no Padinho application port is exposed.
MySQL has no public IP and accepts TCP/3306 only from the bot NSG.

## Bootstrap state storage

Production's committed `backend.tf` intentionally contains an empty OCI
backend. Bootstrap cannot declare that backend until it has created the bucket,
so its declaration is committed as `backend.tf.example`. Copy each `.example`
configuration file without the `.example` suffix when its corresponding step
requires it; the real bootstrap `backend.tf`, all `backend.hcl` files, and all
`terraform.tfvars` files are ignored.

Bootstrap and production remain separate because Terraform must initialize its
backend before it can plan the resources that create that backend. Bootstrap
creates a `NoPublicAccess` bucket named with a generated 32-character
hexadecimal identifier. Object versioning is intentionally disabled. Terraform
prevents accidental destruction of the bucket itself. Bootstrap first uses the
default local backend, then migrates that small state into the bucket it
created:

```sh
cd infrastructure/terraform/bootstrap
terraform init
terraform apply
cp backend.tf.example backend.tf
cp backend.hcl.example backend.hcl
# Replace bucket and namespace with the apply outputs.
terraform init -migrate-state -backend-config=backend.hcl
```

The native OCI backend stores state at the configured key and automatically
creates `<key>.lock` in the same private bucket during state-changing
operations. Both backend files select API-key authentication and
`config_file_profile = "DEFAULT"`; the profile's OCIDs, fingerprint, and private
key path stay in the standard OCI configuration and are never copied into HCL.
Terraform caches backend settings under `.terraform`, which is also ignored.

The similarly named files serve different purposes: `terraform.tfvars` is the
real, automatically loaded input file and stays ignored;
`terraform.tfvars.example` is its committed template. `.terraform.lock.hcl` is
the committed provider dependency lock. The transient state lock is the
`<key>.lock` object in OCI Object Storage, not a local file checked into Git.

## Production

Production resolves availability domain number `3` through OCI instead of
hard-coding a tenancy-specific name. Change `availability_domain_number` when
deploying into a region or tenancy where that number is unavailable.

```sh
cd infrastructure/terraform/environments/production
cp backend.hcl.example backend.hcl
cp terraform.tfvars.example terraform.tfvars
terraform init -backend-config=backend.hcl
terraform plan
terraform apply
```

After a successful apply, copy Terraform's outputs into the ignored Ansible
inventory; the discovered public IP is an output, not a `terraform.tfvars`
input:

```sh
terraform output -raw bot_public_ip
terraform output -raw mysql_private_ip
terraform output -raw mysql_database_name
```

Never commit either copied file. Treat Terraform state as sensitive because the
MySQL administrator password necessarily passes through state. That account is
reserved for provisioning and manual migrations; Padinho receives a distinct
restricted account through Ansible. The public Salada GHCR package is pulled
anonymously, so registry credentials do not belong in Terraform. OCI
credentials should come from the standard OCI config/environment rather than
HCL.
