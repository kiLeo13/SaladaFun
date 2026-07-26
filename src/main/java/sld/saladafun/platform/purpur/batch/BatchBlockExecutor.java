package sld.saladafun.platform.purpur.batch;

import org.bukkit.Material;
import org.bukkit.block.Block;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import org.bukkit.inventory.PlayerInventory;
import org.bukkit.inventory.meta.Damageable;
import org.bukkit.inventory.meta.ItemMeta;
import sld.saladafun.batchbreaking.BatchBlockAction;
import sld.saladafun.batchbreaking.ToolDurabilityMode;

import java.util.Objects;

/**
 * Applies one additional batch break using the configured platform policy.
 */
final class BatchBlockExecutor {

    /**
     * Breaks one additional block.
     *
     * @return whether the platform accepted and performed the break
     */
    boolean breakBlock(
        Player player,
        Block block,
        BatchBlockAction action,
        ToolDurabilityMode durabilityMode
    ) {
        Objects.requireNonNull(player, "player");
        Objects.requireNonNull(block, "block");
        Objects.requireNonNull(action, "action");
        Objects.requireNonNull(durabilityMode, "durabilityMode");

        if (action == BatchBlockAction.NATURAL_DROPS) {
            return block.breakNaturally(true, true);
        }

        ToolDamageSnapshot damage = durabilityMode == ToolDurabilityMode.SINGLE_USE
            ? ToolDamageSnapshot.capture(player.getInventory())
            : null;
        boolean broken = player.breakBlock(block);
        if (broken && damage != null) {
            damage.restore(player.getInventory());
        }
        return broken;
    }

    private record ToolDamageSnapshot(ItemStack item, int damage) {

        private static ToolDamageSnapshot capture(PlayerInventory inventory) {
            ItemStack heldItem = inventory.getItemInMainHand();
            ItemMeta metadata = heldItem.getItemMeta();
            if (!(metadata instanceof Damageable damageable)) {
                return null;
            }
            return new ToolDamageSnapshot(heldItem.clone(), damageable.getDamage());
        }

        private void restore(PlayerInventory inventory) {
            ItemStack current = inventory.getItemInMainHand();
            if (current.getType() == Material.AIR) {
                inventory.setItemInMainHand(item);
                return;
            }
            if (current.getType() != item.getType()) {
                return;
            }
            ItemMeta currentMetadata = current.getItemMeta();
            if (!(currentMetadata instanceof Damageable damageable)) {
                return;
            }
            damageable.setDamage(damage);
            current.setItemMeta(currentMetadata);
            inventory.setItemInMainHand(current);
        }
    }
}
