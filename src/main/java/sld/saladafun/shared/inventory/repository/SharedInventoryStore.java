package sld.saladafun.shared.inventory.repository;

import sld.saladafun.shared.inventory.model.InitialInventoryMode;
import sld.saladafun.shared.inventory.model.InventorySnapshot;
import sld.saladafun.shared.inventory.model.PlayerInventoryBackup;
import sld.saladafun.shared.inventory.model.ReplicaState;
import sld.saladafun.shared.inventory.model.SessionId;
import sld.saladafun.shared.inventory.model.SessionLabel;
import sld.saladafun.shared.inventory.model.SharedInventorySession;

import java.time.LocalDate;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

/**
 * Persistence port for shared sessions, canonical state, backups, and player replicas.
 *
 * <p>Implementations must make each method atomic.</p>
 */
public interface SharedInventoryStore extends AutoCloseable {
    Optional<SharedInventorySession> loadActiveSession();

    SharedInventorySession createSession(
        LocalDate labelDate,
        InitialInventoryMode mode,
        UUID sourcePlayerId,
        InventorySnapshot initialInventory,
        Map<UUID, InventorySnapshot> onlineBackups
    );

    SharedInventorySession resumeSession(
        SessionLabel label,
        Map<UUID, InventorySnapshot> onlineBackups
    );

    void saveCanonical(SessionId sessionId, InventorySnapshot inventory);

    void archive(SessionId sessionId);

    List<SharedInventorySession> listArchivedSessions();

    boolean hasBackup(SessionId sessionId, UUID playerId);

    void saveBackupIfAbsent(
        SessionId sessionId,
        UUID playerId,
        InventorySnapshot personalInventory
    );

    Optional<PlayerInventoryBackup> findBackup(SessionId sessionId, UUID playerId);

    Optional<PlayerInventoryBackup> findPendingRestore(UUID playerId);

    void markRestorePending(SessionId sessionId);

    void markRestored(SessionId sessionId, UUID playerId);

    Optional<ReplicaState> findReplica(SessionId sessionId, UUID playerId);

    void saveReplica(ReplicaState replica);

    @Override
    void close();
}
