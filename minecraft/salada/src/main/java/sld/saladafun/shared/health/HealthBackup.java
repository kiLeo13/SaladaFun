package sld.saladafun.shared.health;

import sld.saladafun.shared.model.RestoreStatus;
import sld.saladafun.shared.model.SessionId;

import java.time.Instant;
import java.util.Objects;
import java.util.UUID;

/** Personal health state captured before a shared session was applied. */
public record HealthBackup(
    SessionId sessionId,
    UUID playerId,
    HealthState state,
    RestoreStatus restoreStatus,
    Instant capturedAt,
    Instant restoredAt
) {
    public HealthBackup {
        Objects.requireNonNull(sessionId, "sessionId");
        Objects.requireNonNull(playerId, "playerId");
        Objects.requireNonNull(state, "state");
        Objects.requireNonNull(restoreStatus, "restoreStatus");
        Objects.requireNonNull(capturedAt, "capturedAt");
    }
}
