# Discord Utils

Discord Utils is a server-side NeoForge 21.1.249 mod for Minecraft 1.21.1.
Its first feature is an optional bidirectional Discord chat bridge with player
join and leave notifications; account linking is deliberately not implemented
yet.

## Requirements

- JDK 21
- NeoForge 21.1.249 on Minecraft 1.21.1

Build and test with the Gradle wrapper:

```text
.\gradlew.bat clean build
```

The deployable artifact is written to `build/libs/`. Install it only on the
dedicated server; clients do not need this server-only mod.

Kotlin for Forge is optional. Discord Utils includes Kotlin 2.2.21 for JDA's
HTTP dependencies when no other mod provides Kotlin. If Kotlin for Forge or
another mod supplies Kotlin 2.2.21 or newer, NeoForge selects that compatible
shared runtime instead. Do not remove Kotlin for Forge when another installed
mod, such as Fzzy Config, declares it as a required dependency.

The build inspects the packaged mod and Jar-in-Jar metadata to ensure its Gradle
placeholders were resolved and the Kotlin fallback remains negotiable,
preventing an invalid mod artifact from reaching the server.

See [`docs/discord-chat.md`](docs/discord-chat.md) for configuration, Discord
permissions, security, and runtime behavior.
