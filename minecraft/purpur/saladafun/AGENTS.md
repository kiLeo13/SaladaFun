# SaladaFun plugin contribution guidance

- Keep this project as one Maven JAR module using the standard project `src/main`
  and `src/test` layout.
- Keep the domain packages under `sld.saladafun.shared` and
  `sld.saladafun.batchbreaking` free of Bukkit, Paper, Purpur, JDBC, jOOQ, and
  SQLite types.
- Keep SQLite/jOOQ implementations under
  `sld.saladafun.persistence.sqlite` and server adapters under
  `sld.saladafun.platform.purpur`.
- Treat `shared-state.db`, the retired `shared-inventory.db`, their WAL/SHM
  companions, and generated `target/` directories as runtime/build output; never
  commit them.
- Run `mvn clean package` from `minecraft/purpur/saladafun` after build or plugin changes.
  From the repository root, use
  `mvn -f minecraft/purpur/saladafun/pom.xml clean package`.
- Deploy only `target/saladafun-1.0.jar`; the `original-` JAR is an unshaded
  intermediate artifact.
