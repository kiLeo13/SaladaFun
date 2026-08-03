package sld.saladafun.shared.effects;

import java.util.Objects;

/** Portable potion-effect value, including all player-visible presentation flags. */
public record EffectState(
    String typeKey,
    int amplifier,
    int durationTicks,
    boolean ambient,
    boolean particles,
    boolean icon,
    EffectState hiddenEffect
) {
    public static final int INFINITE_DURATION = -1;

    public EffectState {
        Objects.requireNonNull(typeKey, "typeKey");
        if (typeKey.isBlank()) {
            throw new IllegalArgumentException("typeKey must not be blank");
        }
        if (amplifier < 0) {
            throw new IllegalArgumentException("amplifier must not be negative");
        }
        if (durationTicks != INFINITE_DURATION && durationTicks < 1) {
            throw new IllegalArgumentException(
                "durationTicks must be positive or INFINITE_DURATION"
            );
        }
        if (hiddenEffect != null && !typeKey.equals(hiddenEffect.typeKey())) {
            throw new IllegalArgumentException(
                "hidden effect layers must have the same type"
            );
        }
    }

    public EffectState(
        String typeKey,
        int amplifier,
        int durationTicks,
        boolean ambient,
        boolean particles,
        boolean icon
    ) {
        this(
            typeKey,
            amplifier,
            durationTicks,
            ambient,
            particles,
            icon,
            null
        );
    }

    public boolean sameDefinition(EffectState other) {
        return other != null
            && typeKey.equals(other.typeKey)
            && amplifier == other.amplifier
            && ambient == other.ambient
            && particles == other.particles
            && icon == other.icon
            && ((hiddenEffect == null && other.hiddenEffect == null)
                || (hiddenEffect != null
                    && hiddenEffect.sameDefinition(other.hiddenEffect)));
    }
}
