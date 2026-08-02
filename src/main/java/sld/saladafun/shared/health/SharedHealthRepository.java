package sld.saladafun.shared.health;

import sld.saladafun.shared.model.InitialStateMode;
import sld.saladafun.shared.model.SessionId;
import sld.saladafun.shared.model.SessionLabel;

import java.time.LocalDate;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

/** Atomic persistence operations required by shared-health lifecycle management. */
public interface SharedHealthRepository {
    Optional<HealthSession> loadActive();

    HealthSession create(
        LocalDate labelDate,
        InitialStateMode mode,
        UUID sourcePlayerId,
        HealthState initial,
        Map<UUID, HealthState> backups
    );

    HealthSession resume(SessionLabel label, Map<UUID, HealthState> backups);

    void saveCanonical(SessionId sessionId, HealthState state);

    void archiveAndMarkRestores(SessionId sessionId);

    List<HealthSession> listArchived();

    void saveBackupIfAbsent(SessionId sessionId, UUID playerId, HealthState state);

    Optional<HealthBackup> findPendingRestore(UUID playerId);

    void markRestored(SessionId sessionId, UUID playerId);
}
