package sld.saladafun.shared.food;

import sld.saladafun.shared.model.InitialStateMode;
import sld.saladafun.shared.model.SessionId;
import sld.saladafun.shared.model.SessionLabel;

import java.time.LocalDate;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

/** Atomic persistence operations required by shared-food lifecycle management. */
public interface SharedFoodRepository {
    Optional<FoodSession> loadActive();

    FoodSession create(
        LocalDate labelDate,
        InitialStateMode mode,
        UUID sourcePlayerId,
        FoodState initial,
        Map<UUID, FoodState> backups
    );

    FoodSession resume(SessionLabel label, Map<UUID, FoodState> backups);

    void saveCanonical(SessionId sessionId, FoodState state);

    void archiveAndMarkRestores(SessionId sessionId);

    List<FoodSession> listArchived();

    void saveBackupIfAbsent(SessionId sessionId, UUID playerId, FoodState state);

    Optional<FoodBackup> findPendingRestore(UUID playerId);

    void markRestored(SessionId sessionId, UUID playerId);
}
