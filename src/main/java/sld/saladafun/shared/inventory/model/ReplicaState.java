package sld.saladafun.shared.inventory.model;

import java.time.Instant;
import java.util.Objects;
import java.util.UUID;

/**
 * Last canonical revision and content fingerprint applied to one Bukkit replica.
 */
public record ReplicaState(
    SessionId sessionId,
    UUID playerId,
    long appliedRevision,
    String inventoryFingerprint,
    Instant observedAt
) {
    public ReplicaState {
        Objects.requireNonNull(sessionId, "sessionId");
        Objects.requireNonNull(playerId, "playerId");
        if (appliedRevision < 0) {
            throw new IllegalArgumentException("appliedRevision must not be negative");
        }
        Objects.requireNonNull(inventoryFingerprint, "inventoryFingerprint");
        Objects.requireNonNull(observedAt, "observedAt");
    }
}
