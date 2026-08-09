package sld.saladafun.shared.health;

import java.util.Comparator;
import java.util.List;
import java.util.Objects;

/** Thread-safe canonical health aggregate with tick-batched delta merging. */
public final class SharedHealth {
    private HealthState state;

    public SharedHealth(HealthState initialState) {
        state = Objects.requireNonNull(initialState, "initialState");
    }

    public synchronized HealthState snapshot() {
        return state;
    }

    /**
     * Adds every accepted player delta, resolves absolute range writes
     * deterministically, clamps once, and publishes at most one revision.
     */
    public synchronized HealthState applyTick(
        List<HealthContribution> contributions,
        boolean lethal
    ) {
        Objects.requireNonNull(contributions, "contributions");
        HealthContribution rangeWinner = contributions.stream()
            .filter(HealthContribution::rangeChanged)
            .max(Comparator.comparing(contribution -> contribution.actorId().toString()))
            .orElse(null);

        double maximumHealth = rangeWinner == null
            ? state.maximumHealth()
            : rangeWinner.observedMaximumHealth();
        double maximumAbsorption = rangeWinner == null
            ? state.maximumAbsorption()
            : rangeWinner.observedMaximumAbsorption();
        double healthDelta = contributions.stream()
            .mapToDouble(HealthContribution::healthDelta)
            .sum();
        double absorptionDelta = contributions.stream()
            .mapToDouble(HealthContribution::absorptionDelta)
            .sum();

        double health = lethal
            ? 0.0
            : clamp(state.health() + healthDelta, 0.0, maximumHealth);
        HealthPhase phase = health == 0.0 ? HealthPhase.DEAD : HealthPhase.ALIVE;
        double absorption = clamp(
            state.absorption() + absorptionDelta,
            0.0,
            maximumAbsorption
        );
        if (phase == HealthPhase.DEAD) {
            absorption = 0.0;
        }

        HealthState candidate = new HealthState(
            health,
            maximumHealth,
            absorption,
            maximumAbsorption,
            phase,
            state.revision()
        );
        if (!sameValues(state, candidate)) {
            state = candidate.withRevision(state.revision() + 1);
        }
        return state;
    }

    /** Revives a dead canonical pool at full red health and no absorption. */
    public synchronized HealthState revive() {
        if (state.phase() == HealthPhase.ALIVE) {
            return state;
        }
        state = new HealthState(
            state.maximumHealth(),
            state.maximumHealth(),
            0.0,
            state.maximumAbsorption(),
            HealthPhase.ALIVE,
            state.revision() + 1
        );
        return state;
    }

    private boolean sameValues(HealthState left, HealthState right) {
        return Double.compare(left.health(), right.health()) == 0
            && Double.compare(left.maximumHealth(), right.maximumHealth()) == 0
            && Double.compare(left.absorption(), right.absorption()) == 0
            && Double.compare(left.maximumAbsorption(), right.maximumAbsorption()) == 0
            && left.phase() == right.phase();
    }

    private double clamp(double value, double minimum, double maximum) {
        return Math.max(minimum, Math.min(value, maximum));
    }
}
