package sld.saladafun.platform.purpur.config;

/** Validated performance settings shared by health, food, and effects. */
public record SharedVitalsSettings(
    int safetyAuditIntervalTicks,
    int persistenceFlushIntervalTicks
) {
    public SharedVitalsSettings {
        if (safetyAuditIntervalTicks < 1) {
            throw new IllegalArgumentException(
                "shared-vitals.safety-audit-interval-ticks must be positive"
            );
        }
        if (persistenceFlushIntervalTicks < 1) {
            throw new IllegalArgumentException(
                "shared-vitals.persistence.flush-interval-ticks must be positive"
            );
        }
    }
}
