package sld.saladafun.shared.health;

/** Immutable canonical shared-health and absorption state. */
public record HealthState(
    double health,
    double maximumHealth,
    double absorption,
    double maximumAbsorption,
    HealthPhase phase,
    long revision
) {
    public HealthState {
        requireFinitePositive(maximumHealth, "maximumHealth");
        requireFiniteNonNegative(maximumAbsorption, "maximumAbsorption");
        requireFiniteInRange(health, 0.0, maximumHealth, "health");
        requireFiniteInRange(absorption, 0.0, maximumAbsorption, "absorption");
        if (phase == null) {
            throw new NullPointerException("phase");
        }
        if ((phase == HealthPhase.DEAD) != (health == 0.0)) {
            throw new IllegalArgumentException("DEAD requires zero health and vice versa");
        }
        if (revision < 0) {
            throw new IllegalArgumentException("revision must not be negative");
        }
    }

    public static HealthState full(double maximumHealth, double maximumAbsorption) {
        return new HealthState(
            maximumHealth,
            maximumHealth,
            0.0,
            maximumAbsorption,
            HealthPhase.ALIVE,
            0
        );
    }

    public HealthState withRevision(long newRevision) {
        return new HealthState(
            health,
            maximumHealth,
            absorption,
            maximumAbsorption,
            phase,
            newRevision
        );
    }

    private static void requireFinitePositive(double value, String name) {
        if (!Double.isFinite(value) || value <= 0.0) {
            throw new IllegalArgumentException(name + " must be finite and positive");
        }
    }

    private static void requireFiniteNonNegative(double value, String name) {
        if (!Double.isFinite(value) || value < 0.0) {
            throw new IllegalArgumentException(name + " must be finite and non-negative");
        }
    }

    private static void requireFiniteInRange(
        double value,
        double minimum,
        double maximum,
        String name
    ) {
        if (!Double.isFinite(value) || value < minimum || value > maximum) {
            throw new IllegalArgumentException(name + " is outside its valid range");
        }
    }
}
