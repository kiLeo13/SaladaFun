package sld.saladafun.shared.inventory.model;

import java.util.Objects;

/**
 * Stable identity for item attributes that determine whether two stacks are compatible.
 */
public record ItemFingerprint(String value) {
    public ItemFingerprint {
        Objects.requireNonNull(value, "value");
        if (value.isBlank()) {
            throw new IllegalArgumentException("Item fingerprints must not be blank");
        }
    }
}
