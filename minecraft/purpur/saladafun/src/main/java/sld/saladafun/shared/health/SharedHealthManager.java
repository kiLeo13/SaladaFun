package sld.saladafun.shared.health;

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

/** Application service owning the optional active shared-health aggregate. */
public final class SharedHealthManager {
    private final SharedHealthRepository repository;
    private final Clock clock;
    private final ZoneId labelZone;
    private HealthSession activeSession;
    private SharedHealth aggregate;

    public SharedHealthManager(
        SharedHealthRepository repository,
        Clock clock,
        ZoneId labelZone
    ) {
        this.repository = Objects.requireNonNull(repository, "repository");
        this.clock = Objects.requireNonNull(clock, "clock");
        this.labelZone = Objects.requireNonNull(labelZone, "labelZone");
    }

    public synchronized void load() {
        activeSession = repository.loadActive().orElse(null);
        aggregate = activeSession == null ? null : new SharedHealth(activeSession.state());
    }

    public synchronized boolean isEnabled() {
        return aggregate != null;
    }

    public synchronized Optional<HealthState> current() {
        return aggregate == null ? Optional.empty() : Optional.of(aggregate.snapshot());
    }

    public synchronized Optional<HealthSession> activeSession() {
        return Optional.ofNullable(activeSession);
    }

    public synchronized HealthSession enableFresh(Map<UUID, HealthState> backups) {
        return enable(
            InitialStateMode.FRESH,
            null,
            HealthState.full(20.0, 0.0),
            backups
        );
    }

    public synchronized HealthSession enableFrom(
        UUID sourcePlayerId,
        HealthState source,
        Map<UUID, HealthState> backups
    ) {
        Objects.requireNonNull(sourcePlayerId, "sourcePlayerId");
        return enable(InitialStateMode.SOURCE_PLAYER, sourcePlayerId, source, backups);
    }

    public synchronized HealthSession resume(
        SessionLabel label,
        Map<UUID, HealthState> backups
    ) {
        requireDisabled();
        activeSession = repository.resume(label, Map.copyOf(backups));
        aggregate = new SharedHealth(activeSession.state());
        return activeSession;
    }

    public synchronized Optional<HealthSession> disable() {
        if (aggregate == null) {
            return Optional.empty();
        }
        repository.saveCanonical(activeSession.id(), aggregate.snapshot());
        repository.archiveAndMarkRestores(activeSession.id());
        HealthSession disabled = activeSession;
        activeSession = null;
        aggregate = null;
        return Optional.of(disabled);
    }

    public synchronized HealthState applyTick(
        List<HealthContribution> contributions,
        boolean lethal
    ) {
        HealthState previous = requireAggregate().snapshot();
        HealthState next = aggregate.applyTick(List.copyOf(contributions), lethal);
        persistOrRollback(previous, next);
        return next;
    }

    public synchronized HealthState revive() {
        HealthState previous = requireAggregate().snapshot();
        HealthState next = aggregate.revive();
        persistOrRollback(previous, next);
        return next;
    }

    public synchronized HealthState join(UUID playerId, HealthState personalState) {
        SharedHealth currentAggregate = requireAggregate();
        repository.saveBackupIfAbsent(
            activeSession.id(),
            playerId,
            personalState.withRevision(0)
        );
        return currentAggregate.snapshot();
    }

    public synchronized Optional<HealthBackup> pendingRestore(UUID playerId) {
        return repository.findPendingRestore(playerId);
    }

    public synchronized void markRestored(HealthBackup backup) {
        repository.markRestored(backup.sessionId(), backup.playerId());
    }

    public synchronized List<HealthSession> archivedSessions() {
        return repository.listArchived();
    }

    private HealthSession enable(
        InitialStateMode mode,
        UUID source,
        HealthState initial,
        Map<UUID, HealthState> backups
    ) {
        requireDisabled();
        activeSession = repository.create(
            LocalDate.now(clock.withZone(labelZone)),
            mode,
            source,
            initial.withRevision(0),
            Map.copyOf(backups)
        );
        aggregate = new SharedHealth(activeSession.state());
        return activeSession;
    }

    private void persistOrRollback(HealthState previous, HealthState next) {
        if (next.revision() == previous.revision()) {
            return;
        }
        try {
            repository.saveCanonical(activeSession.id(), next);
            activeSession = activeSession.withState(next);
        } catch (RuntimeException exception) {
            aggregate = new SharedHealth(previous);
            throw exception;
        }
    }

    private SharedHealth requireAggregate() {
        if (aggregate == null) {
            throw new IllegalStateException("Shared health is disabled");
        }
        return aggregate;
    }

    private void requireDisabled() {
        if (aggregate != null) {
            throw new IllegalStateException("Shared health is already enabled");
        }
    }
}
