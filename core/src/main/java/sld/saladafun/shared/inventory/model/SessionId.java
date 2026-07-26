package sld.saladafun.shared.inventory.model;

import java.util.Objects;
import java.util.UUID;

/**
 * Opaque primary identifier for a shared-inventory session.
 */
public record SessionId(UUID value) {
    public SessionId {
        Objects.requireNonNull(value, "value");
    }

    public static SessionId create() {
        return new SessionId(UUID.randomUUID());
    }
}
