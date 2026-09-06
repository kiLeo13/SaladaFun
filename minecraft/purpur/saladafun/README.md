# SaladaFun

SaladaFun is the SaladaFun ecosystem's Minecraft plugin for Purpur 26.2. It provides
shared player vitals, player-scoped batch breaking, and an optional bidirectional
Discord chat bridge.

## Requirements

- JDK 25
- Maven 3.9 or newer
- Purpur 26.2 for live-server validation

Purpur is a server-provided dependency and is not included in the plugin JAR.
SQLite, jOOQ, JDA, and their required runtime dependencies are shaded into the
deployable artifact.

## Build and test

From this directory:

```text
mvn clean package
```

From the repository root:

```text
mvn -f minecraft/purpur/saladafun/pom.xml clean package
```

The complete unit and SQLite integration test suite runs during the build. The
deployable artifact is `target/saladafun-1.0.jar`; do not deploy the
`target/original-saladafun-1.0.jar` intermediate artifact.

## Documentation

- [`ARCHITECTURE.md`](ARCHITECTURE.md) describes package boundaries, persistence,
  runtime behavior, and verification.
- [`docs/shared-vitals.md`](docs/shared-vitals.md) documents shared health, food,
  and effects.
- [`docs/batch-breaking.md`](docs/batch-breaking.md) documents batch breaking.
- [`docs/discord-chat.md`](docs/discord-chat.md) documents the Discord bridge.
