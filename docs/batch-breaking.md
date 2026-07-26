# Batch breaking

## Commands

```text
/batchbreaking get
/batchbreaking set disabled
/batchbreaking set all
/batchbreaking set <positive-integer>
```

Management permission: `saladafun.batchbreaking.manage`.
Use permission: `saladafun.batchbreaking.use`.

`all` means exact-material matches in chunks sent to the player who broke the
original block. It does not include chunks loaded for other players, unloaded
chunks, or every generated chunk.

A positive integer is an inclusive cube around the original position:

```text
max(abs(dx), abs(dy), abs(dz)) <= range
```

Range one therefore includes the 26 adjacent and diagonal positions. The original
position is excluded because it has already broken normally.

## Safety settings

```yaml
batch-breaking:
  setting: disabled
  additional-block-action: PLAYER_TOOL
  tool-durability: SINGLE_USE
  snapshot-chunks-per-tick: 2
  blocks-per-tick: 64
  max-queued-matches: 8192
```

These settings limit processing rate and memory, not total matches. Large jobs may
take considerable time.

## Additional-block behavior

`additional-block-action` controls how every block after the original is broken:

- `PLAYER_TOOL` calls the player-aware break path. It uses the tool currently in
  the initiating player's main hand, including Silk Touch, Fortune, normal drops,
  experience, and protection-plugin `BlockBreakEvent` checks.
- `NATURAL_DROPS` invokes the platform's natural no-tool break operation. It
  produces the block's tool-independent natural drops and experience, but does not
  create a player `BlockBreakEvent`. It can therefore bypass protection plugins;
  use it only where that behavior is acceptable.
- `NO_DROPS` uses the same player-aware path as `PLAYER_TOOL`, but sets item drops
  off and experience to zero at `LOWEST` event priority. Later plugins remain free
  to cancel or alter the event.

The original block always follows the server's ordinary player-breaking behavior.
The action applies only to additional matched blocks.

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
