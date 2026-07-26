# SaladaFun architecture

## Purpose

SaladaFun is a Java 25 Purpur 26.2 plugin providing a revisioned shared inventory
and player-scoped batch breaking. Domain rules are isolated from Bukkit so they can
be reused by another Minecraft server adapter.

## Modules

The Maven reactor has three modules:

- `core` contains pure Java domain models, synchronized inventory mutations,
  repository ports, join reconciliation, session lifecycle, and cubic range logic.
  It has no Bukkit, jOOQ, JDBC, or SQLite dependency.
- `persistence/sqlite` implements the core persistence port with SQLite JDBC and
  jOOQ. Versioned SQL lives under `db/migration`.
- `platform/purpur` is the plugin artifact. It maps Bukkit objects into core
  snapshots, listens to Purpur events, executes commands, schedules batch work,
  and shades runtime dependencies. Purpur remains a `provided` dependency.

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
2. Captures only sent and still-loaded chunk snapshots under a per-tick budget.
3. Scans immutable snapshots on one background worker.
4. Applies bounded queue backpressure.
5. Revalidates world, player, sent chunk, and exact material on the main thread.
6. Applies one of `PLAYER_TOOL`, `NATURAL_DROPS`, or `NO_DROPS` through a
   dedicated platform executor.

Player-aware generated breaks use a per-player context map as both a recursion
guard and the source for cooperative `LOWEST` event policy. `NO_DROPS` suppresses
item and experience drops there; protection plugins may still cancel or alter the
event later. `NATURAL_DROPS` deliberately uses `Block.breakNaturally` and therefore
does not provide a player event to protection plugins.

Tool durability is independently configured as `SINGLE_USE` or `PER_BLOCK`.
`SINGLE_USE` leaves the original vanilla durability outcome intact and snapshots
the main-hand tool around each successful generated player-aware break, restoring
only its previous damage (or the tool itself when that use would destroy it).
`PER_BLOCK` leaves generated durability changes intact. Natural no-tool breaks
ignore this setting.

Jobs cancel on logout/world change, plugin shutdown, or invalid state. Only one job
may run per player.

## Build and verification

Use JDK 25:

```text
mvn clean package
```

The distributable JAR is `platform/purpur/target/saladafun-purpur-1.0.jar`.
Core concurrency/range tests, real SQLite integration tests, and adapter mapping
tests run in the reactor. Live multi-player Purpur validation remains required
before production rollout because mocks cannot reproduce server event ordering.
