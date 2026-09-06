package sld.saladafun.shared.effects;

import sld.saladafun.shared.model.InitialStateMode;
import sld.saladafun.shared.model.SessionLabel;

import java.time.Clock;
import java.time.LocalDate;
import java.time.ZoneId;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/** Application service owning the optional active shared-effects aggregate. */
public final class SharedEffectsManager {
    private final SharedEffectsRepository repository;
    private final Clock clock;
    private final ZoneId labelZone;
    private EffectsSession activeSession;
    private SharedEffects aggregate;

    public SharedEffectsManager(
        SharedEffectsRepository repository,
        Clock clock,
        ZoneId labelZone
    ) {
        this.repository = Objects.requireNonNull(repository, "repository");
        this.clock = Objects.requireNonNull(clock, "clock");
        this.labelZone = Objects.requireNonNull(labelZone, "labelZone");
    }

    public synchronized void load() {
        activeSession = repository.loadActive().orElse(null);
        aggregate = activeSession == null
            ? null
            : new SharedEffects(activeSession.state());
    }

    public synchronized boolean isEnabled() {
        return aggregate != null;
    }

    public synchronized Optional<EffectsState> current() {
        return aggregate == null ? Optional.empty() : Optional.of(aggregate.snapshot());
    }

    public synchronized Optional<EffectsSession> activeSession() {
        return Optional.ofNullable(activeSession);
    }

    public synchronized EffectsSession enableFresh(
        Map<UUID, EffectsState> backups
    ) {
        return enable(
            InitialStateMode.FRESH,
            null,
            EffectsState.empty(),
            backups
        );
    }

    public synchronized EffectsSession enableFrom(
        UUID sourcePlayerId,
        EffectsState source,
        Map<UUID, EffectsState> backups
    ) {
        Objects.requireNonNull(sourcePlayerId, "sourcePlayerId");
        return enable(
            InitialStateMode.SOURCE_PLAYER,
            sourcePlayerId,
            source,
            backups
        );
    }

    public synchronized EffectsSession resume(
        SessionLabel label,
        Map<UUID, EffectsState> backups
    ) {
        requireDisabled();
        activeSession = repository.resume(label, Map.copyOf(backups));
        aggregate = new SharedEffects(activeSession.state());
        return activeSession;
    }

    public synchronized Optional<EffectsSession> disable() {
        if (aggregate == null) {
            return Optional.empty();
        }
        repository.saveCanonical(activeSession.id(), aggregate.snapshot());
        repository.archiveAndMarkRestores(activeSession.id());
        EffectsSession disabled = activeSession;
        activeSession = null;
        aggregate = null;
        return Optional.of(disabled);
    }

    public synchronized EffectsState applyTick(List<EffectChange> changes) {
        EffectsState previous = requireAggregate().snapshot();
        EffectsState next = aggregate.applyTick(List.copyOf(changes));
        persistOrRollback(previous, next);
        return next;
    }

    public synchronized EffectsState refreshDurations(
        Map<String, EffectState> observed
    ) {
        EffectsState next = requireAggregate().refreshDurations(Map.copyOf(observed));
        repository.saveCanonical(activeSession.id(), next);
        activeSession = activeSession.withState(next);
        return next;
    }

    public synchronized EffectsState join(
        UUID playerId,
        EffectsState personalState
    ) {
        SharedEffects currentAggregate = requireAggregate();
        repository.saveBackupIfAbsent(
            activeSession.id(),
            playerId,
            personalState.withRevision(0)
        );
        return currentAggregate.snapshot();
    }

    public synchronized Optional<EffectsBackup> pendingRestore(UUID playerId) {
        return repository.findPendingRestore(playerId);
    }

    public synchronized void markRestored(EffectsBackup backup) {
        repository.markRestored(backup.sessionId(), backup.playerId());
    }

    public synchronized List<EffectsSession> archivedSessions() {
        return repository.listArchived();
    }

    private EffectsSession enable(
        InitialStateMode mode,
        UUID source,
        EffectsState initial,
        Map<UUID, EffectsState> backups
    ) {
        requireDisabled();
        activeSession = repository.create(
            LocalDate.now(clock.withZone(labelZone)),
            mode,
            source,
            initial.withRevision(0),
            Map.copyOf(backups)
        );
        aggregate = new SharedEffects(activeSession.state());
        return activeSession;
    }

    private void persistOrRollback(EffectsState previous, EffectsState next) {
        if (next.revision() == previous.revision()) {
            return;
        }
        try {
            repository.saveCanonical(activeSession.id(), next);
            activeSession = activeSession.withState(next);
        } catch (RuntimeException exception) {
            aggregate = new SharedEffects(previous);
            throw exception;
        }
    }

    private SharedEffects requireAggregate() {
        if (aggregate == null) {
            throw new IllegalStateException("Shared effects are disabled");
        }
        return aggregate;
    }

    private void requireDisabled() {
        if (aggregate != null) {
            throw new IllegalStateException("Shared effects are already enabled");
        }
    }
}
