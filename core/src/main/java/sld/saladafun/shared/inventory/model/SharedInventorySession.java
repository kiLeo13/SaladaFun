package sld.saladafun.shared.inventory.model;

import java.time.Instant;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/**
 * Persistent metadata and canonical inventory for one activation session.
 */
public record SharedInventorySession(
    SessionId id,
    SessionLabel label,
    SessionStatus status,
    InitialInventoryMode initialMode,
    UUID sourcePlayerId,
    Instant startedAt,
    Instant archivedAt,
    InventorySnapshot inventory
) {
    public SharedInventorySession {
        Objects.requireNonNull(id, "id");
        Objects.requireNonNull(label, "label");
        Objects.requireNonNull(status, "status");
        Objects.requireNonNull(initialMode, "initialMode");
        Objects.requireNonNull(startedAt, "startedAt");
        Objects.requireNonNull(inventory, "inventory");
    }

    public Optional<UUID> sourcePlayer() {
        return Optional.ofNullable(sourcePlayerId);
    }

    public Optional<Instant> archiveTime() {
        return Optional.ofNullable(archivedAt);
    }
}
