# SaladaFun architecture

## Purpose

SaladaFun is a Bukkit/Purpur plugin. The runtime server supplies the Bukkit, Paper,
and Purpur APIs; the plugin JAR must therefore not package those APIs.

## Build

The project uses Maven. `pom.xml` declares Purpur API 26.2 build 2609 as a
`provided` dependency and targets Java 25. Maven plugins are responsible for
compilation, packaging, resource filtering, and test execution:

- `maven-compiler-plugin` compiles with Java release 25.
- `maven-shade-plugin` creates the distributable JAR when runtime dependencies
  are added later.
- `maven-surefire-plugin` runs tests during the Maven test phase.

Build with:

```text
mvn clean package
```

The resulting plugin JAR is written to `target/`.

## IntelliJ IDEA

The project metadata targets an SDK named `25`. If IDEA reports that the SDK is
missing, add `C:\Program Files\Java\jdk-25.0.3` as a Java 25 SDK in the IDE,
then reload the Maven project. The SDK registration is intentionally kept in
IDEA's user settings rather than committed to this project.

Java 25 support requires IntelliJ IDEA 2025.2 or newer. The installed IDEA
2024.1.3 can run the project with Maven, but it may report `JDK_25` as an
unsupported language level; update IDEA before relying on its Java editor and
inspections for Java 25 features.

## Plugin entry point

`src/main/resources/plugin.yml` declares
`sld.saladaFun.SaladaFun` as the runtime entry point. The class currently only
implements the lifecycle hooks and has no external runtime dependencies.
