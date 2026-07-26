package sld.saladafun.shared.inventory.model;

import java.time.Instant;
import java.util.Objects;
import java.util.UUID;

/**
 * Personal inventory captured before a player receives a shared session.
 */
public record PlayerInventoryBackup(
    SessionId sessionId,
    UUID playerId,
    InventorySnapshot inventory,
    RestoreStatus restoreStatus,
    Instant capturedAt,
    Instant restoredAt
) {
    public PlayerInventoryBackup {
        Objects.requireNonNull(sessionId, "sessionId");
        Objects.requireNonNull(playerId, "playerId");
        Objects.requireNonNull(inventory, "inventory");
        Objects.requireNonNull(restoreStatus, "restoreStatus");
        Objects.requireNonNull(capturedAt, "capturedAt");
    }

    public enum RestoreStatus {
        CAPTURED,
        RESTORE_PENDING,
        RESTORED
    }
}
