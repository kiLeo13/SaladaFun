package sld.saladafun.shared.food;

import sld.saladafun.shared.model.InitialStateMode;
import sld.saladafun.shared.model.SessionId;
import sld.saladafun.shared.model.SessionLabel;
import sld.saladafun.shared.model.SessionStatus;

import java.time.Instant;
import java.time.LocalDate;
import java.util.Objects;
import java.util.UUID;

/** Persisted shared-food session. */
public record FoodSession(
    SessionId id,
    SessionLabel label,
    LocalDate sessionDate,
    int dailySequence,
    SessionStatus status,
    InitialStateMode initialMode,
    UUID sourcePlayerId,
    FoodState state,
    Instant startedAt,
    Instant archivedAt
) {
    public FoodSession {
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

    public FoodSession withState(FoodState newState) {
        return new FoodSession(
            id, label, sessionDate, dailySequence, status, initialMode,
            sourcePlayerId, newState, startedAt, archivedAt
        );
    }
}
