package sld.saladafun.platform.purpur.shared;

import org.bukkit.Material;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import org.bukkit.inventory.PlayerInventory;
import sld.saladafun.shared.inventory.model.InventorySnapshot;
import sld.saladafun.shared.inventory.model.ItemFingerprint;
import sld.saladafun.shared.inventory.model.ItemKey;
import sld.saladafun.shared.inventory.model.ItemStackSnapshot;
import sld.saladafun.shared.inventory.model.SlotKey;

import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.ArrayList;
import java.util.EnumMap;
import java.util.HexFormat;
import java.util.List;
import java.util.Map;
import java.util.Optional;

/**
 * Lossless mapping between Purpur inventory objects and the API-independent core model.
 */
public final class PurpurInventoryMapper {
    private static final String PAYLOAD_FORMAT = "purpur-itemstack-nbt-v1";

    public InventorySnapshot snapshot(Player player, long revision) {
        PlayerInventory inventory = player.getInventory();
        var slots = new EnumMap<SlotKey, ItemStackSnapshot>(SlotKey.class);
        for (int index = 0; index < 36; index++) {
            put(slots, fromStorageIndex(index), inventory.getItem(index));
        }
        put(slots, SlotKey.ARMOR_FEET, inventory.getBoots());
        put(slots, SlotKey.ARMOR_LEGS, inventory.getLeggings());
        put(slots, SlotKey.ARMOR_CHEST, inventory.getChestplate());
        put(slots, SlotKey.ARMOR_HEAD, inventory.getHelmet());
        put(slots, SlotKey.OFF_HAND, inventory.getItemInOffHand());
        return new InventorySnapshot(revision, slots);
    }

    public ItemStackSnapshot snapshot(ItemStack item) {
        if (isEmpty(item)) {
            throw new IllegalArgumentException("Cannot snapshot an empty item");
        }
        ItemStack identity = item.clone();
        identity.setAmount(1);
        return new ItemStackSnapshot(
            new ItemKey(item.getType().getKey().toString()),
            new ItemFingerprint(hash(identity.serializeAsBytes())),
            item.getAmount(),
            item.getMaxStackSize(),
            PAYLOAD_FORMAT,
            item.serializeAsBytes()
        );
    }

    public ItemStack toBukkit(ItemStackSnapshot item) {
        ItemStack decoded = ItemStack.deserializeBytes(item.payload());
        decoded.setAmount(item.amount());
        return decoded;
    }

    public List<ItemStackSnapshot> snapshots(List<ItemStack> items) {
        var mapped = new ArrayList<ItemStackSnapshot>();
        for (ItemStack item : items) {
            if (!isEmpty(item)) {
                mapped.add(snapshot(item));
            }
        }
        return List.copyOf(mapped);
    }

    public Optional<SlotKey> fromBukkitSlot(int slot) {
        if (slot >= 0 && slot < 36) {
            return Optional.of(fromStorageIndex(slot));
        }
        return switch (slot) {
            case 36 -> Optional.of(SlotKey.ARMOR_FEET);
            case 37 -> Optional.of(SlotKey.ARMOR_LEGS);
            case 38 -> Optional.of(SlotKey.ARMOR_CHEST);
            case 39 -> Optional.of(SlotKey.ARMOR_HEAD);
            case 40 -> Optional.of(SlotKey.OFF_HAND);
            default -> Optional.empty();
        };
    }

    public int toBukkitSlot(SlotKey slot) {
        String name = slot.name();
        if (name.startsWith("HOTBAR_")) {
            return Integer.parseInt(name.substring("HOTBAR_".length()));
        }
        if (name.startsWith("MAIN_")) {
            return 9 + Integer.parseInt(name.substring("MAIN_".length()));
        }
        return switch (slot) {
            case ARMOR_FEET -> 36;
            case ARMOR_LEGS -> 37;
            case ARMOR_CHEST -> 38;
            case ARMOR_HEAD -> 39;
            case OFF_HAND -> 40;
            default -> throw new IllegalArgumentException("Unsupported slot " + slot);
        };
    }

    public ItemStackSnapshot nullableSnapshot(ItemStack item) {
        return isEmpty(item) ? null : snapshot(item);
    }

    private SlotKey fromStorageIndex(int index) {
        return index < 9
            ? SlotKey.valueOf("HOTBAR_" + index)
            : SlotKey.valueOf("MAIN_" + (index - 9));
    }

    private void put(
        Map<SlotKey, ItemStackSnapshot> target,
        SlotKey slot,
        ItemStack item
    ) {
        if (!isEmpty(item)) {
            target.put(slot, snapshot(item));
        }
    }

    private boolean isEmpty(ItemStack item) {
        return item == null || item.getType() == Material.AIR || item.getAmount() < 1;
    }

    private String hash(byte[] bytes) {
        try {
            return HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(bytes));
        } catch (NoSuchAlgorithmException impossible) {
            throw new IllegalStateException("SHA-256 is required by the Java platform", impossible);
        }
    }
}
