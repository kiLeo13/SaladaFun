package sld.saladafun.shared.food;

import sld.saladafun.shared.model.RestoreStatus;
import sld.saladafun.shared.model.SessionId;

import java.time.Instant;
import java.util.Objects;
import java.util.UUID;

/** Personal food state captured before a shared session was applied. */
public record FoodBackup(
    SessionId sessionId,
    UUID playerId,
    FoodState state,
    RestoreStatus restoreStatus,
    Instant capturedAt,
    Instant restoredAt
) {
    public FoodBackup {
        Objects.requireNonNull(sessionId, "sessionId");
        Objects.requireNonNull(playerId, "playerId");
        Objects.requireNonNull(state, "state");
        Objects.requireNonNull(restoreStatus, "restoreStatus");
        Objects.requireNonNull(capturedAt, "capturedAt");
    }
}
