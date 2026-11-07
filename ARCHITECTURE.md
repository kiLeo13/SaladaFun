# SaladaFun ecosystem architecture

## Purpose

SaladaFun is a monorepo for independently deployable applications used by one
Discord guild and its game servers. Projects share ownership and conventions
without being forced into the same language or build system.

## Repository structure

See [README.md](README.md#repository-layout) for the directory layout.

The root owns ecosystem governance, navigation, architecture, shared schema,
and infrastructure. Each deployable project owns its implementation, tests,
build, operational documentation, and detailed architecture. There is no root
language-specific aggregator.

Shared migrations are a cross-release contract owned by the independent root
`database` Go module. Its self-contained Linux executable is built and run
manually on the Padinho VM; application code and Compose never own schema
evolution.

Terraform owns OCI resources; Ansible configures the resulting VM and
reconciles Docker Compose. GitHub Actions tests and publishes Padinho's image to
the public GitHub Container Registry package used by Compose.
Networking and provisioning requirements are documented in
[Terraform's runbook](infrastructure/terraform/README.md).

## Projects

### Minecraft: SaladaFun

`minecraft/purpur/saladafun` is a Java 25/Purpur 26.2 plugin providing shared player
vitals, batch breaking, and an optional Discord chat bridge. See
[`minecraft/purpur/saladafun/ARCHITECTURE.md`](minecraft/purpur/saladafun/ARCHITECTURE.md).

### Minecraft: Discord Utils

`minecraft/neoforge/discord-utils` is a Java 21/NeoForge 21.1.249 server-side
Discord utility mod. Its first feature is the optional bidirectional chat bridge.

### Discord: Padinho

Padinho, under `discord/padinho`, is a Go 1.26 application designed for a 1 GB
`VM.Standard.E2.1.Micro`.

Birthday scheduling, movie tracking, and other feature boundaries are documented
in [Padinho architecture](discord/padinho/ARCHITECTURE.md).

Project build commands are listed in [README.md](README.md#verification).
