# SaladaFun architecture

## Purpose

SaladaFun is a Java 25 Purpur 26.2 plugin providing a revisioned shared inventory
and player-scoped batch breaking. Domain rules are isolated from Bukkit so they can
be reused by another Minecraft server adapter.

## Source layout

SaladaFun is one Maven JAR project using the standard root source layout:

- `src/main/java/sld/saladafun/shared` and
  `src/main/java/sld/saladafun/batchbreaking` contain pure Java domain models,
  synchronized inventory mutations, repository ports, join reconciliation,
  session lifecycle, and cubic range logic. These packages have no Bukkit, jOOQ,
  JDBC, or SQLite dependency.
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

Java packages retain the collision-safe `sld.saladafun` root but are feature-first,
for example `sld.saladafun.shared.inventory` and
`sld.saladafun.batchbreaking`.

## Shared inventory

### Slots

The canonical inventory includes 36 storage/hotbar slots, four armor slots, and
off-hand. It excludes the selected hotbar index, cursor, crafting grid, Ender
Chest, experience, health, hunger, and effects.

Items cross the core boundary as immutable snapshots containing a namespaced item
key, stack-compatibility fingerprint, amount, maximum size, payload format, and
opaque bytes. Purpur currently uses versioned `ItemStack` NBT bytes. Core logic is
portable; persisted payloads require an adapter that understands their format.

### Authority and revisions

`SharedInventory` is the runtime authority and has synchronized, in-memory
methods. It never calls Bukkit or persistence while holding its monitor.

- Accepted ordinary changes use compare-and-set against the old canonical slot.
- Accepted changes receive a monotonically increasing global revision.
- Drops, pickups, and death clearing first reserve slots or capacity.
- Reservations make quantities unavailable immediately and are committed or
  rolled back after the final event outcome.
- Operation UUIDs provide idempotency.
- Online Bukkit inventories are replicas, not independent authorities.

`SharedInventoryManager` owns zero or one aggregate. It is constructed once by the
plugin composition root and injected; there is no static singleton or service
locator. `current()` returns `Optional`, never `null`.

### Join reconciliation

There is intentionally no consistency configuration. Bukkit does not expose an
offline inventory API, so reconciliation always occurs when an inventory becomes
available on join (and for already-online players after a plugin reload).

1. A player without a session backup is backed up and receives canonical state.
2. Returning replicas are compared with their last applied revision and fingerprint.
3. An older or unmarked replica receives database state.
4. A content change whose applied revision is at least canonical is promoted as a
   new LWW revision.

This treats proven newer Bukkit state as authoritative without allowing an old
offline copy to resurrect stale shared items.

### Lifecycle

`/shared inventory enable` creates an empty session. Supplying an online player
creates a session from that player's inventory. Activation closes open inventories,
persists all online personal backups and the session transactionally, then applies
canonical state.

Disabling persists and archives canonical state, marks backups pending, restores
online players, and restores offline players on their next join. Resume uses an
archived session's canonical items and resets replica markers.

Session UUIDs are primary keys. Labels are generated transactionally in local
server time as `yyyyMMdd_nn`; timestamps remain UTC.

### Event cooperation

Handlers that make decisions run at `LOWEST`. Cancellable operations use a
read-only `MONITOR` handler to commit or roll back after other plugins have made
their decisions. MONITOR handlers never mutate Bukkit events.

### Death

`FOLLOW_GAMERULE`, `DROPS_ON_DEATH`, `FADES_ON_DEATH`, and `KEEPS_ON_DEATH`
are described in `docs/shared-inventory.md`. The core serializes global clearing
so two deaths cannot drop the same canonical items. Optional
`getItemsToKeep()` contents are reinserted after clearing.

## Persistence

`shared-inventory.db` uses foreign keys, WAL, a five-second busy timeout, and FULL
synchronous durability. Migrations are checksum-protected.

Main tables:

- `shared_inventory_control`
- `shared_inventory_session`
- `shared_inventory_slot`
- `player_inventory_backup`
- `player_inventory_backup_slot`
- `player_inventory_replica`
- `schema_history`

Lifecycle operations and canonical snapshots use jOOQ transactions. SQLite and
Minecraft player/world files cannot share one atomic transaction; revisions,
fail-closed startup, backup-first activation, and join reconciliation minimize that
unavoidable crash boundary.

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
JDA and Bukkit types do not enter the shared-inventory or batch-breaking domain
packages.

Minecraft-to-Discord traffic observes final uncancelled `AsyncChatEvent` messages
at `MONITOR`, converts the Adventure component to plain text, and sends it through
a JDA `IncomingWebhookClient`. Webhook messages use the player's username, a
UUID-addressed MC Heads avatar, and an empty allowed-mentions set. JDA owns rate
limiting and asynchronous HTTP execution.

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
without interrupting active traffic. Plugin shutdown closes both.

Operational setup, security requirements, supported message content, and live
verification are documented in `docs/discord-chat.md`.

## Build and verification

Use JDK 25:

```text
mvn clean package
```

The distributable JAR is `target/saladafun-1.0.jar`. Domain concurrency/range
tests, real SQLite integration tests, and adapter mapping tests run in the same
Maven build. Live multi-player Purpur validation remains required before
production rollout because mocks cannot reproduce server event ordering.
