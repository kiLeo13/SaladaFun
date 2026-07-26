# Project contribution guidance

- Keep the Maven build configured for Java 25 and Purpur 26.2.
- Keep the project as one Maven JAR module using the standard root `src/main`
  and `src/test` layout.
- Keep the Purpur API dependency scoped as `provided`; the server supplies it at runtime.
- Keep the domain packages under `sld.saladafun.shared` and
  `sld.saladafun.batchbreaking` free of Bukkit, Paper, Purpur, JDBC, jOOQ, and
  SQLite types.
- Keep SQLite/jOOQ implementations under
  `sld.saladafun.persistence.sqlite` and server adapters under
  `sld.saladafun.platform.purpur`.
- Keep `plugin.yml`, `config.yml`, and database migrations under
  `src/main/resources`.
- Treat `shared-inventory.db`, its WAL/SHM companions, and generated `target/`
  directories as runtime/build output; never commit them.
- Run `mvn clean package` after build or plugin changes.
- Deploy only `target/saladafun-1.0.jar`; the `original-` JAR is an unshaded
  intermediate artifact.
- Add or update tests for behavior changes and update `ARCHITECTURE.md` when the
  build or runtime structure changes.
- Do not commit IDE user settings or generated `target/` output.
