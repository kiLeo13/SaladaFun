package sld.saladafun.shared.inventory.model;

import java.util.Arrays;
import java.util.Objects;

/**
 * Immutable platform-neutral item stack.
 *
 * <p>The payload is deliberately opaque to the core. A platform adapter owns its encoding
 * and records that encoding in {@link #payloadFormat()}.</p>
 */
public final class ItemStackSnapshot {
    private final ItemKey itemKey;
    private final ItemFingerprint fingerprint;
    private final int amount;
    private final int maximumStackSize;
    private final String payloadFormat;
    private final byte[] payload;

    public ItemStackSnapshot(
        ItemKey itemKey,
        ItemFingerprint fingerprint,
        int amount,
        int maximumStackSize,
        String payloadFormat,
        byte[] payload
    ) {
        this.itemKey = Objects.requireNonNull(itemKey, "itemKey");
        this.fingerprint = Objects.requireNonNull(fingerprint, "fingerprint");
        if (amount < 1) {
            throw new IllegalArgumentException("amount must be positive");
        }
        if (maximumStackSize < 1 || amount > maximumStackSize) {
            throw new IllegalArgumentException("amount exceeds the maximum stack size");
        }
        this.amount = amount;
        this.maximumStackSize = maximumStackSize;
        this.payloadFormat = Objects.requireNonNull(payloadFormat, "payloadFormat");
        if (payloadFormat.isBlank()) {
            throw new IllegalArgumentException("payloadFormat must not be blank");
        }
        this.payload = Objects.requireNonNull(payload, "payload").clone();
    }

    public ItemKey itemKey() {
        return itemKey;
    }

    public ItemFingerprint fingerprint() {
        return fingerprint;
    }

    public int amount() {
        return amount;
    }

    public int maximumStackSize() {
        return maximumStackSize;
    }

    public String payloadFormat() {
        return payloadFormat;
    }

    public byte[] payload() {
        return payload.clone();
    }

    public ItemStackSnapshot withAmount(int newAmount) {
        return new ItemStackSnapshot(
            itemKey, fingerprint, newAmount, maximumStackSize, payloadFormat, payload
        );
    }

    public boolean canStackWith(ItemStackSnapshot other) {
        return other != null
            && itemKey.equals(other.itemKey)
            && fingerprint.equals(other.fingerprint)
            && maximumStackSize == other.maximumStackSize;
    }

    @Override
    public boolean equals(Object candidate) {
        if (this == candidate) {
            return true;
        }
        if (!(candidate instanceof ItemStackSnapshot other)) {
            return false;
        }
        return amount == other.amount
            && maximumStackSize == other.maximumStackSize
            && itemKey.equals(other.itemKey)
            && fingerprint.equals(other.fingerprint)
            && payloadFormat.equals(other.payloadFormat)
            && Arrays.equals(payload, other.payload);
    }

    @Override
    public int hashCode() {
        int result = Objects.hash(itemKey, fingerprint, amount, maximumStackSize, payloadFormat);
        return 31 * result + Arrays.hashCode(payload);
    }

    @Override
    public String toString() {
        return itemKey.value() + " x" + amount;
    }
}
