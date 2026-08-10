# SaladaFun ecosystem architecture

## Purpose

SaladaFun is a monorepo for independently deployable applications used by one
Discord guild and its game servers. Projects share ownership and conventions
without being forced into the same language or build system.

## Repository structure

```text
.
|-- database/                 Ecosystem SQL migration history
|-- infrastructure/
|   |-- ansible/              VM configuration and Compose reconciliation
|   `-- terraform/            OCI bootstrap, modules, and production stack
|-- discord/
|   `-- padinho/              Padinho Discord bot
|-- minecraft/
|   `-- salada/               Purpur plugin
|-- AGENTS.md
|-- ARCHITECTURE.md
`-- README.md
```

The root owns ecosystem governance, navigation, architecture, shared schema,
and infrastructure. Each deployable project owns its implementation, tests,
build, operational documentation, and detailed architecture. There is no root
language-specific aggregator.

Shared migrations are a cross-release contract. Padinho copies them into its
image and applies them before connecting to Discord. Terraform owns OCI
resources; Ansible configures the resulting VM and reconciles Docker Compose.

## Projects

### Minecraft: Salada

`minecraft/salada` is a Java 25/Purpur 26.2 plugin providing shared player
vitals, batch breaking, and an optional Discord chat bridge. See
[`minecraft/salada/ARCHITECTURE.md`](minecraft/salada/ARCHITECTURE.md).

### Discord: Padinho

Padinho, under `discord/padinho`, is a Go 1.26 application designed for a 1 GB
`VM.Standard.E2.1.Micro`. Its command declarations are independent of DiscordGo;
a frozen registry produces immutable Discord definitions and runtime dispatch.
Middleware composes from registry to route, and application data lives in a
typed request rather than context values. See
[`discord/padinho/ARCHITECTURE.md`](discord/padinho/ARCHITECTURE.md).

## Verification

Projects remain independent:

```text
mvn -f minecraft/salada/pom.xml clean package
cd discord/padinho && go test -race ./...
```
