# SaladaFun architecture

## Purpose

SaladaFun is a Java 25 Purpur 26.2 plugin providing shared player vitals,
player-scoped batch breaking, and a Discord chat bridge. Domain rules are isolated
from Bukkit so they can be reused by another Minecraft server adapter.

## Source layout

SaladaFun is one Maven JAR project using the standard root source layout:

- `src/main/java/sld/saladafun/shared` and
  `src/main/java/sld/saladafun/batchbreaking` contain pure Java domain models and
  cubic range logic. These packages have no Bukkit, jOOQ, JDBC, or SQLite
  dependency.
- `src/main/java/sld/saladafun/persistence/sqlite` implements the persistence
  port with SQLite JDBC and jOOQ.
- `src/main/java/sld/saladafun/platform/purpur` maps Bukkit objects into domain
  snapshots, listens to Purpur events, executes commands, schedules batch work,
  and adapts JDA events for the optional Discord chat bridge. Purpur remains a
  `provided` dependency.
- `src/main/resources` contains `plugin.yml`, `config.yml`, and versioned SQL
  under `db/migration`.
- `src/test/java` mirrors the production package hierarchy.

Maven compiles these packages together and shades required runtime dependencies
into one deployable plugin JAR. Package boundaries keep responsibilities clear
without introducing separate Maven modules.

Java packages retain the collision-safe `sld.saladafun` root and are
feature-first, for example `sld.saladafun.batchbreaking`.

## Shared health, food, and effects

Health, food, and effects are independent session modules under
`sld.saladafun.shared`.
Their aggregates, lifecycle managers, models, and repository interfaces have no
Bukkit, Paper, Purpur, JDBC, jOOQ, or SQLite dependency.

Players are replicas of one in-memory canonical state per enabled module. At
`ServerTickEndEvent`, the Purpur adapters reconcile only players marked dirty by
accepted gameplay events. Idle ticks do no polling or aggregate work. A periodic
full safety audit catches direct API changes for which Bukkit exposes no event.
Current health, absorption, food level, saturation, and exhaustion changes merge
as additive deltas. Each aggregate sums the complete tick and clamps once, then
emits at most one revision.

Maximum health and maximum absorption are absolute ranges rather than deltas.
Same-tick conflicts use deterministic UUID-ordered LWW. The Purpur adapter
preserves existing attribute bases and modifiers and installs one namespaced,
transient additive override that makes the effective range canonical. Personal
restoration removes only that override. Effect reconciliation precedes health
reconciliation, so canonical potion attribute modifiers are visible to the health
range mapper in the same tick.

Effects are canonical per namespaced type. Different types merge, while
same-type conflicts use UUID-ordered LWW. The domain model preserves amplifier,
remaining duration, ambient/particle/icon flags, and recursive hidden-effect
chains without exposing Bukkit types. Natural duration countdown does not create
a gameplay revision every tick; the safety audit refreshes the durable remaining
duration. The platform adapter guards generated events against feedback.

A definitive, non-cancelled `PlayerDeathEvent` latches the tick as lethal.
Lethality dominates same-tick healing, clears canonical health and absorption,
and causes one guarded fan-out to other living online players. The adapter retains
the primary Bukkit `DamageSource` and uses it for every generated secondary death;
generated events cannot recursively replace that source or start another wave.
If summed nonlethal deltas empty the pool without a primary death event, the last
accepted player damage event in that tick supplies the shared source.
The first post-respawn event revives the pool at full canonical range. Dead
players are not marked restored until a post-respawn or subsequent join makes
their Bukkit state writable.

`SharedHealthManager`, `SharedFoodManager`, and `SharedEffectsManager` own session
lifecycle. SQLite implementations use normalized typed health, food, and effect
tables in `shared-state.db`; no generic nullable state blob or inventory schema is
retained. Personal backups, active controls, archived sessions, and module-scoped
`yyyyMMdd_nn` labels are transactionally persisted.

Canonical hot-path saves pass through one coalescing single-writer queue. The
queue keeps SQLite transactions and FULL-synchronous disk flushes off the server
thread and retains only the latest pending save per module. Lifecycle operations
and clean shutdown use synchronous flush barriers for ordering. Bukkit and Purpur
objects never cross the asynchronous boundary; only immutable domain snapshots
do. Audit and flush intervals come from `shared-vitals` configuration at startup.

Commands are routed through `SharedCommand` to module-specific delegates. Health,
food, and effects can be enabled, disabled, inspected, and resumed independently. Detailed
gameplay and operational behavior is documented in `docs/shared-vitals.md`.
Lifecycle listeners use `LOWEST`; effects reconcile at `LOWEST` and health/food
reconcile at `LOW`, allowing later
plugins to cancel or adjust ordinary gameplay without SaladaFun claiming final
event authority.

## Batch breaking

Settings are `disabled`, `all`, or a positive integer. Positive values use
inclusive Chebyshev distance, producing a full cube; range one includes all 26
neighbors around the original block. `all` means chunks sent to the initiating
player, never every generated or globally loaded chunk.

The Purpur adapter:

1. Observes the original final `BlockBreakEvent`.
2. Treats accepted water and lava bucket fills as same-fluid removal triggers.
3. Optionally treats final player-caused animal damage as a same-species trigger.
4. Restricts finite jobs to intersecting sent chunks and exact cubic coordinates.
5. In `ASYNC`, scans immutable snapshots on one worker and applies internal,
   bounded main-thread batches. In `SYNC`, completes the job in one deliberately
   blocking main-thread operation.
6. Revalidates world, player, sent chunk, and exact material on the main thread.
7. Applies one of `PLAYER_TOOL`, `NATURAL_DROPS`, or `NO_DROPS` through a
   dedicated platform executor.

Player-aware generated breaks use a per-player context map as both a recursion
guard and the source for cooperative `LOWEST` event policy. `NO_DROPS` suppresses
item and experience drops at `HIGHEST` and cancels the final generated
`BlockDropItemEvent`; protection plugins may still cancel the break.
`NATURAL_DROPS` deliberately uses `Block.breakNaturally` and therefore does not
provide a player event to protection plugins.

Generated player-aware breaks restore the player's pre-break food level,
saturation, and exhaustion in a `finally` block, including when another plugin
cancels the break or an exception occurs.

Tool durability is independently configured as `SINGLE_USE` or `PER_BLOCK`.
`SINGLE_USE` leaves the original vanilla durability outcome intact and snapshots
the main-hand tool around each successful generated player-aware break, restoring
only its previous damage (or the tool itself when that use would destroy it).
`PER_BLOCK` leaves generated durability changes intact. Natural no-tool breaks
ignore this setting.

Jobs cancel on logout/world change, plugin shutdown, or invalid state. Only one job
may run per player. Fluid removals emit cancellable generated block-break events,
and generated animal damage is recursion-guarded. Automatic completion is silent
in player chat; diagnostic completion counts are logged only at `FINE`.

`/saladafun reloadconfig` parses and validates the on-disk YAML before calling the
platform reload API. The shared mutable `PluginSettings` view is replaced only
after successful validation, so registered listeners and commands observe new
settings without reconstruction.

## Discord chat bridge

The optional bridge is entirely under `sld.saladafun.platform.purpur.discord`;
JDA and Bukkit types do not enter the shared-vitals or batch-breaking domain
packages.

Minecraft-to-Discord traffic observes final uncancelled `AsyncChatEvent` messages
at `MONITOR`, converts the Adventure component to plain text, and sends it through
a JDA `IncomingWebhookClient`. Webhook messages use the player's username, a
UUID-addressed MC Heads avatar with the vanilla outer head layer, and an empty
allowed-mentions set. JDA owns rate limiting and asynchronous HTTP execution.

Discord-to-Minecraft traffic uses a `createLight` JDA session with only
`GUILD_MESSAGES` and privileged `MESSAGE_CONTENT`. Optional cache flags, member
cache, and guild chunking remain disabled. The listener accepts only the
configured channel, rejects bot and webhook authors to prevent bridge loops, and
copies `getContentDisplay()`, image-attachment count, sticker count, and effective
Discord user name into a JDA-free record. The resulting Adventure broadcast is
scheduled on the Minecraft thread. Image/sticker counters appear in light purple
on the line after message text.

`DiscordChatBridge` owns at most one active and one inactive candidate session.
Configuration reloads stage the candidate until Discord READY confirms that the
configured guild message channel is visible. A successful candidate atomically
replaces and closes the old session; a failed or superseded candidate is closed
without interrupting active traffic. Plugin shutdown closes both and waits for
JDA termination before Purpur unloads the plugin classes.

Operational setup, security requirements, supported message content, and live
verification are documented in `docs/discord-chat.md`.

## Build and verification

Use JDK 25:

```text
mvn clean package
```

The distributable JAR is `target/saladafun-1.0.jar`. Domain merge/range tests,
real SQLite integration tests, and Purpur adapter tests run in the same Maven
build. Live multi-player Purpur validation remains required before production
rollout because mocks cannot reproduce server event ordering.
