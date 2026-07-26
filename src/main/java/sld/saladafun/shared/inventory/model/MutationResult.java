package sld.saladafun.shared.inventory.model;

import java.util.Objects;

/**
 * Mutation outcome together with the authoritative state the caller should apply.
 */
public record MutationResult(
    MutationStatus status,
    InventorySnapshot snapshot,
    int acceptedAmount,
    int remainingAmount
) {
    public MutationResult {
        Objects.requireNonNull(status, "status");
        Objects.requireNonNull(snapshot, "snapshot");
        if (acceptedAmount < 0 || remainingAmount < 0) {
            throw new IllegalArgumentException("amounts must not be negative");
        }
    }

    public boolean accepted() {
        return status == MutationStatus.ACCEPTED;
    }
}
