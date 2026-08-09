# SaladaFun ecosystem architecture

## Purpose

SaladaFun is a monorepo for independently deployable applications used by one
Discord guild and its game servers. Keeping the projects together makes shared
ownership, conventions, and cross-project documentation straightforward without
forcing every application into the same language or build system.

## Repository structure

Projects are grouped first by platform and then by deployable application:

```text
.
├── minecraft/
│   ├── AGENTS.md
│   └── salada/
│       ├── AGENTS.md
│       ├── ARCHITECTURE.md
│       ├── README.md
│       ├── docs/
│       ├── pom.xml
│       └── src/
├── AGENTS.md
├── ARCHITECTURE.md
└── README.md
```

The repository root contains ecosystem-wide governance, navigation, and
architecture. A platform directory may define shared conventions for its child
projects. Each deployable project owns its build configuration, implementation,
tests, operational documentation, and detailed architecture.

There is intentionally no root Maven aggregator. This keeps Salada independently
buildable and leaves room for Discord bots, web applications, and infrastructure
to use the tools appropriate to them.

## Projects

### Minecraft: Salada

`minecraft/salada` is a Java 25/Purpur 26.2 plugin providing shared player
vitals, batch breaking, and an optional Discord chat bridge. It remains one Maven
JAR project with isolated domain, SQLite persistence, and Purpur adapter
boundaries.

See [`minecraft/salada/ARCHITECTURE.md`](minecraft/salada/ARCHITECTURE.md) for its
detailed runtime and package architecture.

## Build and verification

Projects are verified independently. Build Salada from the repository root with:

```text
mvn -f minecraft/salada/pom.xml clean package
```

Its deployable artifact is
`minecraft/salada/target/saladafun-1.0.jar`.
