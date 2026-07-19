# Project contribution guidance

- Keep the Maven build configured for Java 25 and Purpur 26.2.
- Keep the Purpur API dependency scoped as `provided`; the server supplies it at runtime.
- Run `mvn clean package` after build or plugin changes.
- Add or update tests for behavior changes and update `ARCHITECTURE.md` when the
  build or runtime structure changes.
- Do not commit IDE user settings or generated `target/` output.
