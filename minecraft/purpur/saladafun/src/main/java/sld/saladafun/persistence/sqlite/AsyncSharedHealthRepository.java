package sld.saladafun.persistence.sqlite;

import sld.saladafun.shared.health.HealthBackup;
import sld.saladafun.shared.health.HealthSession;
import sld.saladafun.shared.health.HealthState;
import sld.saladafun.shared.health.SharedHealthRepository;
import sld.saladafun.shared.model.InitialStateMode;
import sld.saladafun.shared.model.SessionId;
import sld.saladafun.shared.model.SessionLabel;

import java.time.LocalDate;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/** Moves canonical health writes off-thread while serializing lifecycle barriers. */
public final class AsyncSharedHealthRepository implements SharedHealthRepository {
    private final SharedHealthRepository delegate;
    private final CoalescingPersistenceWriter writer;

    public AsyncSharedHealthRepository(
        SharedHealthRepository delegate,
        CoalescingPersistenceWriter writer
    ) {
        this.delegate = Objects.requireNonNull(delegate, "delegate");
        this.writer = Objects.requireNonNull(writer, "writer");
    }

    @Override
    public Optional<HealthSession> loadActive() {
        barrier();
        return delegate.loadActive();
    }

    @Override
    public HealthSession create(
        LocalDate labelDate,
        InitialStateMode mode,
        UUID sourcePlayerId,
        HealthState initial,
        Map<UUID, HealthState> backups
    ) {
        barrier();
        return delegate.create(labelDate, mode, sourcePlayerId, initial, backups);
    }

    @Override
    public HealthSession resume(
        SessionLabel label,
        Map<UUID, HealthState> backups
    ) {
        barrier();
        return delegate.resume(label, backups);
    }

    @Override
    public void saveCanonical(SessionId sessionId, HealthState state) {
        writer.submitLatest(
            "health:" + sessionId.value(),
            () -> delegate.saveCanonical(sessionId, state)
        );
    }

    @Override
    public void archiveAndMarkRestores(SessionId sessionId) {
        barrier();
        delegate.archiveAndMarkRestores(sessionId);
    }

    @Override
    public List<HealthSession> listArchived() {
        barrier();
        return delegate.listArchived();
    }

    @Override
    public void saveBackupIfAbsent(
        SessionId sessionId,
        UUID playerId,
        HealthState state
    ) {
        barrier();
        delegate.saveBackupIfAbsent(sessionId, playerId, state);
    }

    @Override
    public Optional<HealthBackup> findPendingRestore(UUID playerId) {
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
