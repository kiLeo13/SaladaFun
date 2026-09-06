# Purpur contribution guidance

These rules apply to every project under `minecraft/purpur/` and supplement the
repository and Minecraft guidance.

- Keep the shared platform baseline at Java 25 and Purpur 26.2 unless an
  explicitly documented ecosystem migration changes it.
- Scope the Purpur API as `provided`; the server supplies it at runtime, so it
  must not be shaded into deployable plugin JARs.
- Keep Bukkit, Paper, and Purpur types at project platform boundaries and out of
  reusable domain code.
- Keep `plugin.yml`, default configuration, and any bundled migrations in the
  standard project resources directory.
- Verify behavior changes with the affected project's complete build and
  document any live Purpur checks that mocks cannot reproduce.
