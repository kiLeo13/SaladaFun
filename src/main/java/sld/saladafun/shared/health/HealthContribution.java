package sld.saladafun.shared.health;

import java.util.Objects;
import java.util.UUID;

/** One player's observed contribution to a shared-health tick. */
public record HealthContribution(
    UUID actorId,
    double healthDelta,
    double absorptionDelta,
    boolean rangeChanged,
    double observedMaximumHealth,
    double observedMaximumAbsorption
) {
    public HealthContribution {
        Objects.requireNonNull(actorId, "actorId");
        if (!Double.isFinite(healthDelta) || !Double.isFinite(absorptionDelta)) {
            throw new IllegalArgumentException("health deltas must be finite");
        }
        if (!Double.isFinite(observedMaximumHealth) || observedMaximumHealth <= 0.0) {
            throw new IllegalArgumentException("observedMaximumHealth must be positive");
        }
        if (!Double.isFinite(observedMaximumAbsorption)
            || observedMaximumAbsorption < 0.0) {
            throw new IllegalArgumentException(
                "observedMaximumAbsorption must be non-negative"
            );
        }
    }
}
