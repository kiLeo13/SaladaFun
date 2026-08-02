# Shared health and food

## Commands

```text
/shared health enable [online-source-player]
/shared health disable
/shared health status
/shared health resume <yyyyMMdd_nn>

/shared food enable [online-source-player]
/shared food disable
/shared food status
/shared food resume <yyyyMMdd_nn>
```

Enabling without a source creates a fresh state. Health starts at 20/20 with no
absorption; food starts at level 20, saturation 5, and exhaustion 0. Health and
food are independent and may be enabled at the same time.

Permissions are `saladafun.shared.health.manage` and
`saladafun.shared.food.manage`, both operator-only by default.

## Tick-batched delta merge

Player values are replicas of one canonical state. At the end of each server tick,
the adapter compares each online player with the last state SaladaFun applied. All
differences are added to the prior canonical value and clamped once.

For a shared food level of 10, a same-tick loss of 1 and gain of 4 produce 13:

```text
10 + (-1 + 4) = 13
```

This is an additive delta merge, not last-write-wins over absolute current values.
It intentionally means one running player drains the shared food pool and several
running players drain it faster. Synchronization writes are guarded and do not
feed back as player contributions.

Food level, saturation, and exhaustion are all canonical. Sharing only the visible
food level would leave hidden metabolism state inconsistent.

## Health ranges and absorption

Current red health and current absorption merge as additive deltas. The effective
`MAX_HEALTH` and `MAX_ABSORPTION` ranges are canonical absolute values. If a
player's effective maximum health becomes 4 points, every participant receives a
4-point range at the tick boundary.

Conflicting range changes in one tick use deterministic LWW: the proposal belonging
to the lexicographically greatest player UUID wins. Range writes are clamped before
the merged current amounts are published.

SaladaFun applies a namespaced transient attribute modifier to reach the canonical
effective range. Existing bases, equipment modifiers, potion modifiers, and other
plugin modifiers are left installed. The override is replaced on synchronization
and removed when personal health is restored.

Potion/status effects themselves remain personal. Their resulting red-health,
absorption, and range changes participate in the shared state. This avoids trying
to clone effect duration, amplifier, particles, source, and cure semantics across
players while still sharing the vital outcome.

## Death and respawn

A real `PlayerDeathEvent` makes the entire tick lethal. Same-tick healing cannot
rescue the pool after that definitive death. At tick end, SaladaFun sets canonical
health and absorption to zero and kills every other online living participant.
Generated secondary deaths are guarded against starting another wave.

The first post-respawn event revives the canonical pool at its full current maximum
health and zero absorption. Later respawning players receive the latest canonical
state. Inventory behavior is entirely vanilla or owned by other plugins; SaladaFun
does not contain a shared-inventory feature.

## Sessions, restoration, and storage

Each module owns an independent active session and archive. Labels use
`yyyyMMdd_nn`, with daily sequences scoped to the module. Personal values are
captured before canonical state is applied. Disabling restores loaded living
players immediately; dead players retain a pending restore until respawn or join.

`shared-state.db` uses SQLite foreign keys, WAL, a five-second busy timeout, FULL
synchronous durability, and checksum-protected migrations. Canonical changes are
persisted before the next replica broadcast. If persistence fails, the manager
rolls its in-memory candidate back rather than accepting an undurable revision.

The retired `shared-inventory.db` is never opened, modified, migrated, or deleted.

## Live verification

Before production use, verify with at least two real clients:

1. Lose one food point while another player eats four points in the same tick.
2. Apply simultaneous damage and healing.
3. Change one player's maximum-health and maximum-absorption attributes.
4. Gain and consume absorption through effects.
5. Trigger a lethal tick while another player heals.
6. Disable each module while one participant is dead, then respawn.
7. Restart with both modules active and resume archived sessions.
