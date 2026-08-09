package sld.saladafun.shared.effects;

import java.util.Map;
import java.util.Objects;

/** Immutable canonical potion-effect map keyed by namespaced effect type. */
public record EffectsState(Map<String, EffectState> effects, long revision) {
    public EffectsState {
        Objects.requireNonNull(effects, "effects");
        effects = Map.copyOf(effects);
        effects.forEach((key, effect) -> {
            if (!key.equals(effect.typeKey())) {
                throw new IllegalArgumentException(
                    "effect map key must match EffectState.typeKey"
                );
            }
        });
        if (revision < 0) {
            throw new IllegalArgumentException("revision must not be negative");
        }
    }

    public static EffectsState empty() {
        return new EffectsState(Map.of(), 0);
    }

    public EffectsState withRevision(long newRevision) {
        return new EffectsState(effects, newRevision);
    }
}
