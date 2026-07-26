package sld.saladafun.platform.purpur.shared;

import org.bukkit.Server;
import org.bukkit.entity.Player;
import sld.saladafun.shared.inventory.SharedInventoryManager;
import sld.saladafun.shared.inventory.model.InventorySnapshot;
import sld.saladafun.shared.inventory.model.SlotKey;

import java.util.HashSet;
import java.util.Objects;
import java.util.Set;
import java.util.UUID;

/**
 * Applies canonical snapshots to Bukkit replicas while suppressing feedback events.
 */
public final class PlayerInventorySynchronizer {
    private final Server server;
    private final SharedInventoryManager manager;
    private final PurpurInventoryMapper mapper;
    private final Set<UUID> synchronizing = new HashSet<>();

    public PlayerInventorySynchronizer(
        Server server,
        SharedInventoryManager manager,
        PurpurInventoryMapper mapper
    ) {
        this.server = Objects.requireNonNull(server, "server");
        this.manager = Objects.requireNonNull(manager, "manager");
        this.mapper = Objects.requireNonNull(mapper, "mapper");
    }

    public boolean isSynchronizing(Player player) {
        return synchronizing.contains(player.getUniqueId());
    }

    public void apply(Player player, InventorySnapshot snapshot) {
        int heldSlot = player.getInventory().getHeldItemSlot();
        synchronizing.add(player.getUniqueId());
        try {
            for (SlotKey slot : SlotKey.values()) {
                player.getInventory().setItem(
                    mapper.toBukkitSlot(slot),
                    snapshot.item(slot).map(mapper::toBukkit).orElse(null)
                );
            }
            player.getInventory().setHeldItemSlot(heldSlot);
            player.updateInventory();
            manager.markReplicaApplied(player.getUniqueId(), snapshot);
        } finally {
            synchronizing.remove(player.getUniqueId());
        }
    }

    public void applyToAll(InventorySnapshot snapshot) {
        for (Player player : server.getOnlinePlayers()) {
            apply(player, snapshot);
        }
    }
}
