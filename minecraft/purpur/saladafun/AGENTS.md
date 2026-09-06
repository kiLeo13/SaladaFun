# SaladaFun plugin contribution guidance

- Keep the Maven build configured for Java 25 and Purpur 26.2.
- Keep this project as one Maven JAR module using the standard project `src/main`
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
- Treat `shared-state.db`, the retired `shared-inventory.db`, their WAL/SHM
  companions, and generated `target/` directories as runtime/build output; never
  commit them.
- Run `mvn clean package` from `minecraft/purpur/saladafun` after build or plugin changes.
  From the repository root, use
  `mvn -f minecraft/purpur/saladafun/pom.xml clean package`.
- Deploy only `target/saladafun-1.0.jar`; the `original-` JAR is an unshaded
  intermediate artifact.
- Add or update tests for behavior changes and update `ARCHITECTURE.md` when the
  project build or runtime structure changes.
- Do not commit IDE user settings or generated `target/` output.
