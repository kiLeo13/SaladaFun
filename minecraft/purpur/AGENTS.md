# Purpur contribution guidance

These rules apply to every project under `minecraft/purpur/` and supplement the
repository and Minecraft guidance.

- Keep the shared platform baseline at Java 25 and Purpur 26.2 unless an
  explicitly documented ecosystem migration changes it.
- Scope the Purpur API as `provided`; the server supplies it at runtime, so it
  must not be shaded into deployable plugin JARs.
- Verify behavior changes with the affected project's complete build and
  document any live Purpur checks that mocks cannot reproduce.
