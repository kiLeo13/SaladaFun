package sld.saladafun.shared.effects;

import sld.saladafun.shared.model.RestoreStatus;
import sld.saladafun.shared.model.SessionId;

import java.time.Instant;
import java.util.Objects;
import java.util.UUID;

/** Personal effect map captured before a shared session was applied. */
public record EffectsBackup(
    SessionId sessionId,
    UUID playerId,
    EffectsState state,
    RestoreStatus restoreStatus,
    Instant capturedAt,
    Instant restoredAt
) {
    public EffectsBackup {
        Objects.requireNonNull(sessionId, "sessionId");
        Objects.requireNonNull(playerId, "playerId");
        Objects.requireNonNull(state, "state");
        Objects.requireNonNull(restoreStatus, "restoreStatus");
        Objects.requireNonNull(capturedAt, "capturedAt");
    }
}
