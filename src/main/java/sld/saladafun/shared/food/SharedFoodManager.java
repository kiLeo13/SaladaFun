package sld.saladafun.shared.food;

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

/** Application service owning the optional active shared-food aggregate. */
public final class SharedFoodManager {
    private final SharedFoodRepository repository;
    private final Clock clock;
    private final ZoneId labelZone;
    private FoodSession activeSession;
    private SharedFood aggregate;

    public SharedFoodManager(
        SharedFoodRepository repository,
        Clock clock,
        ZoneId labelZone
    ) {
        this.repository = Objects.requireNonNull(repository, "repository");
        this.clock = Objects.requireNonNull(clock, "clock");
        this.labelZone = Objects.requireNonNull(labelZone, "labelZone");
    }

    public synchronized void load() {
        activeSession = repository.loadActive().orElse(null);
        aggregate = activeSession == null ? null : new SharedFood(activeSession.state());
    }

    public synchronized boolean isEnabled() {
        return aggregate != null;
    }

    public synchronized Optional<FoodState> current() {
        return aggregate == null ? Optional.empty() : Optional.of(aggregate.snapshot());
    }

    public synchronized Optional<FoodSession> activeSession() {
        return Optional.ofNullable(activeSession);
    }

    public synchronized FoodSession enableFresh(Map<UUID, FoodState> backups) {
        return enable(InitialStateMode.FRESH, null, FoodState.fresh(), backups);
    }

    public synchronized FoodSession enableFrom(
        UUID sourcePlayerId,
        FoodState source,
        Map<UUID, FoodState> backups
    ) {
        Objects.requireNonNull(sourcePlayerId, "sourcePlayerId");
        return enable(InitialStateMode.SOURCE_PLAYER, sourcePlayerId, source, backups);
    }

    public synchronized FoodSession resume(
        SessionLabel label,
        Map<UUID, FoodState> backups
    ) {
        requireDisabled();
        activeSession = repository.resume(label, Map.copyOf(backups));
        aggregate = new SharedFood(activeSession.state());
        return activeSession;
    }

    public synchronized Optional<FoodSession> disable() {
        if (aggregate == null) {
            return Optional.empty();
        }
        repository.saveCanonical(activeSession.id(), aggregate.snapshot());
        repository.archiveAndMarkRestores(activeSession.id());
        FoodSession disabled = activeSession;
        activeSession = null;
        aggregate = null;
        return Optional.of(disabled);
    }

    public synchronized FoodState applyTick(List<FoodContribution> contributions) {
        FoodState previous = requireAggregate().snapshot();
        FoodState next = aggregate.applyTick(List.copyOf(contributions));
        persistOrRollback(previous, next);
        return next;
    }

    public synchronized FoodState join(UUID playerId, FoodState personalState) {
        SharedFood currentAggregate = requireAggregate();
        repository.saveBackupIfAbsent(
            activeSession.id(),
            playerId,
            personalState.withRevision(0)
        );
        return currentAggregate.snapshot();
    }

    public synchronized Optional<FoodBackup> pendingRestore(UUID playerId) {
        return repository.findPendingRestore(playerId);
    }

    public synchronized void markRestored(FoodBackup backup) {
        repository.markRestored(backup.sessionId(), backup.playerId());
    }

    public synchronized List<FoodSession> archivedSessions() {
        return repository.listArchived();
    }

    private FoodSession enable(
        InitialStateMode mode,
        UUID source,
        FoodState initial,
        Map<UUID, FoodState> backups
    ) {
        requireDisabled();
        activeSession = repository.create(
            LocalDate.now(clock.withZone(labelZone)),
            mode,
            source,
            initial.withRevision(0),
            Map.copyOf(backups)
        );
        aggregate = new SharedFood(activeSession.state());
        return activeSession;
    }

    private void persistOrRollback(FoodState previous, FoodState next) {
        if (next.revision() == previous.revision()) {
            return;
        }
        try {
            repository.saveCanonical(activeSession.id(), next);
            activeSession = activeSession.withState(next);
        } catch (RuntimeException exception) {
            aggregate = new SharedFood(previous);
            throw exception;
        }
    }

    private SharedFood requireAggregate() {
        if (aggregate == null) {
            throw new IllegalStateException("Shared food is disabled");
        }
        return aggregate;
    }

    private void requireDisabled() {
        if (aggregate != null) {
            throw new IllegalStateException("Shared food is already enabled");
        }
    }
}
