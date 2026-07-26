# Batch breaking

## Commands

```text
/batchbreaking get
/batchbreaking set disabled
/batchbreaking set all
/batchbreaking set <positive-integer>
/saladafun reloadconfig
```

Management permission: `saladafun.batchbreaking.manage`.
Use permission: `saladafun.batchbreaking.use`.
Reload permission: `saladafun.reloadconfig`.

`/saladafun reloadconfig` validates `plugins/SaladaFun/config.yml` before
replacing the settings shared by the active listeners and commands. Invalid YAML
or option values leave the previous live settings in place. An already-running
batch keeps the settings captured when it started; subsequent jobs use the
reloaded values.

`all` means exact-material matches in chunks sent to the player who broke the
original block. It does not include chunks loaded for other players, unloaded
chunks, or every generated chunk.

A positive integer is an inclusive cube around the original position:

```text
max(abs(dx), abs(dy), abs(dz)) <= range
```

Range one therefore includes the 26 adjacent and diagonal positions. The original
position is excluded because it has already broken normally.

Finite ranges inspect only chunks intersecting the cube and only coordinates
inside it. They do not scan unrelated sent chunks or the rest of the world height.

## Triggers

An allowed player action starts a batch after its event survives cancellation:

- Breaking a block finds blocks of the exact same material.
- Filling a bucket from a water source finds other water blocks. Additional water
  is removed without drops after a generated `BlockBreakEvent` is accepted.
- With `include-animals: true`, fatally damaging an `Animals` entity directly or
  with a projectile finds the same entity type. For example, killing a pig targets
  pigs, not all animal types. Generated damage still uses the player as its source,
  allowing ordinary damage/death listeners, drops, and experience behavior.

Animal `all` scope is limited to chunks sent to the initiating player. Finite
animal ranges use the same inclusive cube as blocks.

Automatic batches are silent in player chat. Explicit `/batchbreaking` command
responses remain visible.

## Execution

```yaml
batch-breaking:
  setting: disabled
  additional-block-action: PLAYER_TOOL
  tool-durability: SINGLE_USE
  sync-batching: ASYNC
  include-animals: false
```

- `ASYNC` snapshots relevant sent chunks on the main thread, scans them on one
  background worker, and applies changes in fast internal main-thread batches.
  Bukkit world and entity mutation never occurs off-thread.
- `SYNC` scans and applies the complete job in one main-thread operation. It can
  freeze the server for a large range or `all`, but the entire result is visible
  when the server resumes. It does not schedule continuation tasks.

Internal queue and per-tick limits are intentionally not administrator settings.

## Additional-block behavior

`additional-block-action` controls how every block after the original is broken:

- `PLAYER_TOOL` calls the player-aware break path. It uses the tool currently in
  the initiating player's main hand, including Silk Touch, Fortune, normal drops,
  experience, and protection-plugin `BlockBreakEvent` checks.
- `NATURAL_DROPS` invokes the platform's natural no-tool break operation. It
  produces the block's tool-independent natural drops and experience, but does not
  create a player `BlockBreakEvent`. It can therefore bypass protection plugins;
  use it only where that behavior is acceptable.
- `NO_DROPS` uses the same player-aware path as `PLAYER_TOOL`, enforces zero
  experience at the final mutable break-event priority, and cancels the generated
  `BlockDropItemEvent`. It therefore produces neither item nor experience drops.

The original block always follows the server's ordinary player-breaking behavior.
The action applies only to additional matched solid blocks. Water and animal
batches use their trigger-specific behavior described above.

## Tool durability

`tool-durability` applies to the player-aware `PLAYER_TOOL` and `NO_DROPS`
actions:

- `SINGLE_USE` is the default. The original ordinary break keeps its normal
  durability outcome, including Unbreaking, while generated breaks have their
  durability changes neutralized. A generated break that would destroy the tool
  restores the pre-break tool.
- `PER_BLOCK` leaves every generated break's normal durability cost in place. A
  job stops benefiting from a tool when it breaks; subsequent blocks use whatever
  the player then holds.

`NATURAL_DROPS` does not use a player tool, so `tool-durability` has no effect in
that mode. Because jobs may span many ticks, `PLAYER_TOOL` and `NO_DROPS` use the
player's main-hand item at the time each additional block is processed.

Every generated player-aware block break snapshots and restores food level,
saturation, and exhaustion. The original manual break remains vanilla, but the
additional batch cannot make the player hungry.
