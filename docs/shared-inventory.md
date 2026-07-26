# Shared inventory

## Commands

```text
/shared inventory enable
/shared inventory enable <online-source-player>
/shared inventory disable
/shared inventory status
/shared inventory resume <yyyyMMdd_nn>
```

Enabling without a source always starts empty. It never chooses a player randomly.
Archived sessions remain available through `resume`.

Permission: `saladafun.shared.inventory.manage` (operators by default).

## Configuration

```yaml
shared-inventory:
  death-behavior: FOLLOW_GAMERULE
  respect-items-to-keep: true
```

There is no optional consistency mode. Every loaded player inventory is reconciled
on join, and already-online players are reconciled after plugin reload.

### Death behaviors

- `FOLLOW_GAMERULE` follows the final effective keep-inventory result. False
  produces ordinary death loss; true retains canonical state.
- `DROPS_ON_DEATH` proposes `keepInventory = false`. The shared inventory is lost
  globally and Bukkit produces one normal set of drops.
- `FADES_ON_DEATH` clears the shared inventory globally and removes its normal
  death drops, permanently destroying the items.
- `KEEPS_ON_DEATH` retains canonical state and suppresses shared death drops.

With `respect-items-to-keep: true`, Paper's final `getItemsToKeep()` list is
inserted back into canonical state after a death clear. With `false`, retained
items are excluded and the respawned player is overwritten from canonical state.

`DROPS_ON_DEATH` and gamerule-driven drops let one participant lose the inventory
for everyone. Administrators should treat that as an intentional gameplay rule and
potential griefing surface.

## Recovery and data

Personal inventories are captured before a shared inventory is applied. Never edit
or delete `shared-inventory.db`, `shared-inventory.db-wal`, or
`shared-inventory.db-shm` while the server is running.

On database migration, checksum, or write failure, the plugin fails closed instead
of guessing which inventory is authoritative.
