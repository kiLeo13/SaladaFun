package sld.saladafun.persistence.sqlite;

import sld.saladafun.shared.effects.EffectsBackup;
import sld.saladafun.shared.effects.EffectsSession;
import sld.saladafun.shared.effects.EffectsState;
import sld.saladafun.shared.effects.SharedEffectsRepository;
import sld.saladafun.shared.model.InitialStateMode;
import sld.saladafun.shared.model.SessionId;
import sld.saladafun.shared.model.SessionLabel;

import java.time.LocalDate;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/** Moves canonical effect writes off-thread while serializing lifecycle barriers. */
public final class AsyncSharedEffectsRepository implements SharedEffectsRepository {
    private final SharedEffectsRepository delegate;
    private final CoalescingPersistenceWriter writer;

    public AsyncSharedEffectsRepository(
        SharedEffectsRepository delegate,
        CoalescingPersistenceWriter writer
    ) {
        this.delegate = Objects.requireNonNull(delegate, "delegate");
        this.writer = Objects.requireNonNull(writer, "writer");
    }

    @Override
    public Optional<EffectsSession> loadActive() {
        barrier();
        return delegate.loadActive();
    }

    @Override
    public EffectsSession create(
        LocalDate labelDate,
        InitialStateMode mode,
        UUID sourcePlayerId,
        EffectsState initial,
        Map<UUID, EffectsState> backups
    ) {
        barrier();
        return delegate.create(labelDate, mode, sourcePlayerId, initial, backups);
    }

    @Override
    public EffectsSession resume(
        SessionLabel label,
        Map<UUID, EffectsState> backups
    ) {
        barrier();
        return delegate.resume(label, backups);
    }

    @Override
    public void saveCanonical(SessionId sessionId, EffectsState state) {
        writer.submitLatest(
            "effects:" + sessionId.value(),
            () -> delegate.saveCanonical(sessionId, state)
        );
    }

    @Override
    public void archiveAndMarkRestores(SessionId sessionId) {
        barrier();
        delegate.archiveAndMarkRestores(sessionId);
    }

    @Override
    public List<EffectsSession> listArchived() {
        barrier();
        return delegate.listArchived();
    }

    @Override
    public void saveBackupIfAbsent(
        SessionId sessionId,
        UUID playerId,
        EffectsState state
    ) {
        barrier();
        delegate.saveBackupIfAbsent(sessionId, playerId, state);
    }

    @Override
    public Optional<EffectsBackup> findPendingRestore(UUID playerId) {
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
