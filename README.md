# SaladaFun

SaladaFun is a monorepo for the applications and integrations used by a private
Discord guild and its game servers. Projects share repository-wide standards and
documentation while remaining independently buildable and deployable.

## Repository layout

```text
.
├── minecraft/
│   └── salada/       Java 25/Purpur 26.2 Minecraft plugin
├── AGENTS.md         Repository-wide contribution guidance
└── ARCHITECTURE.md   Ecosystem structure and project index
```

Platform-wide guidance lives inside each platform directory. Project-specific
source code, tests, build files, operational documentation, and architecture stay
inside the corresponding project directory.

## Projects

| Project | Description | Documentation |
| --- | --- | --- |
| Salada | Minecraft gameplay features and an optional Discord chat bridge | [`minecraft/salada`](minecraft/salada/README.md) |

## Build the Minecraft plugin

JDK 25 and Maven are required.

```text
mvn -f minecraft/salada/pom.xml clean package
```

The deployable plugin is
`minecraft/salada/target/saladafun-1.0.jar`.

## License

This repository does not currently include a license. Add an explicit license
before distributing or accepting contributions under defined reuse terms.
