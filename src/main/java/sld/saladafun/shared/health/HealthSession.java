package sld.saladafun.shared.health;

import sld.saladafun.shared.model.InitialStateMode;
import sld.saladafun.shared.model.SessionId;
import sld.saladafun.shared.model.SessionLabel;
import sld.saladafun.shared.model.SessionStatus;

import java.time.Instant;
import java.time.LocalDate;
import java.util.Objects;
import java.util.UUID;

/** Persisted shared-health session. */
public record HealthSession(
    SessionId id,
    SessionLabel label,
    LocalDate sessionDate,
    int dailySequence,
    SessionStatus status,
    InitialStateMode initialMode,
    UUID sourcePlayerId,
    HealthState state,
    Instant startedAt,
    Instant archivedAt
) {
    public HealthSession {
        Objects.requireNonNull(id, "id");
        Objects.requireNonNull(label, "label");
        Objects.requireNonNull(sessionDate, "sessionDate");
        Objects.requireNonNull(status, "status");
        Objects.requireNonNull(initialMode, "initialMode");
        Objects.requireNonNull(state, "state");
        Objects.requireNonNull(startedAt, "startedAt");
        if (dailySequence < 1) {
            throw new IllegalArgumentException("dailySequence must be positive");
        }
    }

    public HealthSession withState(HealthState newState) {
        return new HealthSession(
            id, label, sessionDate, dailySequence, status, initialMode,
            sourcePlayerId, newState, startedAt, archivedAt
        );
    }
}
