package sld.saladafun.shared.inventory.model;

import java.util.Objects;
import java.util.UUID;

/**
 * Idempotency key for a proposed inventory mutation.
 */
public record OperationId(UUID value) {
    public OperationId {
        Objects.requireNonNull(value, "value");
    }

    public static OperationId create() {
        return new OperationId(UUID.randomUUID());
    }
}
