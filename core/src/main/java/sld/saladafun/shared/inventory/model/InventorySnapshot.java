package sld.saladafun.shared.inventory.model;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.Collections;
import java.util.EnumMap;
import java.util.HexFormat;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;

/**
 * Immutable snapshot of all shared slots at a particular canonical revision.
 */
public final class InventorySnapshot {
    private final long revision;
    private final Map<SlotKey, ItemStackSnapshot> slots;

    public InventorySnapshot(long revision, Map<SlotKey, ItemStackSnapshot> slots) {
        if (revision < 0) {
            throw new IllegalArgumentException("revision must not be negative");
        }
        this.revision = revision;
        var copy = new EnumMap<SlotKey, ItemStackSnapshot>(SlotKey.class);
        copy.putAll(Objects.requireNonNull(slots, "slots"));
        this.slots = Collections.unmodifiableMap(copy);
    }

    public static InventorySnapshot empty(long revision) {
        return new InventorySnapshot(revision, Map.of());
    }

    public long revision() {
        return revision;
    }

    public Map<SlotKey, ItemStackSnapshot> slots() {
        return slots;
    }

    public Optional<ItemStackSnapshot> item(SlotKey slot) {
        return Optional.ofNullable(slots.get(Objects.requireNonNull(slot, "slot")));
    }

    /**
     * Returns a deterministic content fingerprint independent of the revision number.
     */
    public String fingerprint() {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            for (SlotKey slot : SlotKey.values()) {
                digest.update(slot.name().getBytes(StandardCharsets.UTF_8));
                ItemStackSnapshot item = slots.get(slot);
                if (item != null) {
                    digest.update(item.itemKey().value().getBytes(StandardCharsets.UTF_8));
                    digest.update(item.fingerprint().value().getBytes(StandardCharsets.UTF_8));
                    digest.update(Integer.toString(item.amount()).getBytes(StandardCharsets.UTF_8));
                    digest.update(item.payload());
                }
            }
            return HexFormat.of().formatHex(digest.digest());
        } catch (NoSuchAlgorithmException impossible) {
            throw new IllegalStateException("SHA-256 is required by the Java platform", impossible);
        }
    }
}
