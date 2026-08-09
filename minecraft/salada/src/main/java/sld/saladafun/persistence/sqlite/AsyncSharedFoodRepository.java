package sld.saladafun.persistence.sqlite;

import sld.saladafun.shared.food.FoodBackup;
import sld.saladafun.shared.food.FoodSession;
import sld.saladafun.shared.food.FoodState;
import sld.saladafun.shared.food.SharedFoodRepository;
import sld.saladafun.shared.model.InitialStateMode;
import sld.saladafun.shared.model.SessionId;
import sld.saladafun.shared.model.SessionLabel;

import java.time.LocalDate;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/** Moves canonical food writes off-thread while serializing lifecycle barriers. */
public final class AsyncSharedFoodRepository implements SharedFoodRepository {
    private final SharedFoodRepository delegate;
    private final CoalescingPersistenceWriter writer;

    public AsyncSharedFoodRepository(
        SharedFoodRepository delegate,
        CoalescingPersistenceWriter writer
    ) {
        this.delegate = Objects.requireNonNull(delegate, "delegate");
        this.writer = Objects.requireNonNull(writer, "writer");
    }

    @Override
    public Optional<FoodSession> loadActive() {
        barrier();
        return delegate.loadActive();
    }

    @Override
    public FoodSession create(
        LocalDate labelDate,
        InitialStateMode mode,
        UUID sourcePlayerId,
        FoodState initial,
        Map<UUID, FoodState> backups
    ) {
        barrier();
        return delegate.create(labelDate, mode, sourcePlayerId, initial, backups);
    }

    @Override
    public FoodSession resume(SessionLabel label, Map<UUID, FoodState> backups) {
        barrier();
        return delegate.resume(label, backups);
    }

    @Override
    public void saveCanonical(SessionId sessionId, FoodState state) {
        writer.submitLatest(
            "food:" + sessionId.value(),
            () -> delegate.saveCanonical(sessionId, state)
        );
    }

    @Override
    public void archiveAndMarkRestores(SessionId sessionId) {
        barrier();
        delegate.archiveAndMarkRestores(sessionId);
    }

    @Override
    public List<FoodSession> listArchived() {
        barrier();
        return delegate.listArchived();
    }

    @Override
    public void saveBackupIfAbsent(
        SessionId sessionId,
        UUID playerId,
        FoodState state
    ) {
        barrier();
        delegate.saveBackupIfAbsent(sessionId, playerId, state);
    }

    @Override
    public Optional<FoodBackup> findPendingRestore(UUID playerId) {
        barrier();
        return delegate.findPendingRestore(playerId);
    }

    @Override
    public void markRestored(SessionId sessionId, UUID playerId) {
        barrier();
        delegate.markRestored(sessionId, playerId);
    }

    private void barrier() {
        writer.flush();
    }
}
