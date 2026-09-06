# Discord Utils

Discord Utils is a server-side NeoForge 21.1.249 mod for Minecraft 1.21.1.
Its first feature is an optional bidirectional Discord chat bridge; account
linking is deliberately not implemented yet.

## Requirements

- JDK 21
- NeoForge 21.1.249 on Minecraft 1.21.1

Build and test with the Gradle wrapper:

```text
.\gradlew.bat clean build
```

The deployable artifact is written to `build/libs/`. Install it only on the
dedicated server; clients do not need this server-only mod.

The build also inspects the packaged `META-INF/neoforge.mods.toml` to ensure its
Gradle placeholders were resolved, preventing an invalid mod artifact from
reaching the server.

See [`docs/discord-chat.md`](docs/discord-chat.md) for configuration, Discord
permissions, security, and runtime behavior.
