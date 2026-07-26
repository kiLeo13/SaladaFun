package sld.saladafun.shared.inventory.model;

import java.util.Objects;

/**
 * Platform-neutral namespaced item identifier, for example {@code minecraft:stone}.
 */
public record ItemKey(String value) {
    public ItemKey {
        Objects.requireNonNull(value, "value");
        if (value.isBlank() || !value.contains(":")) {
            throw new IllegalArgumentException("Item keys must be non-blank and namespaced");
        }
    }
}
