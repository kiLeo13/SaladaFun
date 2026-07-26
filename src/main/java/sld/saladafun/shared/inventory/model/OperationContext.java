package sld.saladafun.shared.inventory.model;

import java.util.Objects;
import java.util.UUID;

/**
 * Audit context for an inventory mutation.
 */
public record OperationContext(UUID actorId, String operationType) {
    public OperationContext {
        Objects.requireNonNull(actorId, "actorId");
        Objects.requireNonNull(operationType, "operationType");
        if (operationType.isBlank()) {
            throw new IllegalArgumentException("operationType must not be blank");
        }
    }
}
