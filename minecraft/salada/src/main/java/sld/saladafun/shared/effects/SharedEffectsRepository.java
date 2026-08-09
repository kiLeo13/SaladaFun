package sld.saladafun.shared.effects;

import sld.saladafun.shared.model.InitialStateMode;
import sld.saladafun.shared.model.SessionId;
import sld.saladafun.shared.model.SessionLabel;

import java.time.LocalDate;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

/** Atomic persistence operations required by shared-effects lifecycle management. */
public interface SharedEffectsRepository {
    Optional<EffectsSession> loadActive();

    EffectsSession create(
        LocalDate labelDate,
        InitialStateMode mode,
        UUID sourcePlayerId,
        EffectsState initial,
        Map<UUID, EffectsState> backups
    );

    EffectsSession resume(SessionLabel label, Map<UUID, EffectsState> backups);

    void saveCanonical(SessionId sessionId, EffectsState state);

    void archiveAndMarkRestores(SessionId sessionId);

    List<EffectsSession> listArchived();

    void saveBackupIfAbsent(
        SessionId sessionId,
        UUID playerId,
        EffectsState state
    );

    Optional<EffectsBackup> findPendingRestore(UUID playerId);

    void markRestored(SessionId sessionId, UUID playerId);
}
