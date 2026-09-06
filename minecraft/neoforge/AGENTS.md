# NeoForge contribution guidance

- Keep projects on Java 21, Minecraft 1.21.1, and NeoForge 21.1.249 unless an
  explicitly documented migration changes the platform baseline.
- Use the Gradle wrapper for all builds and keep NeoForge-provided APIs out of
  JDA-facing domain classes.
- Package non-Minecraft runtime libraries through NeoForge Jar-in-Jar and test
  the packaged artifact on a dedicated server.
- Keep client-only code absent from these server-only projects and declare
  server-only compatibility in `neoforge.mods.toml`.
