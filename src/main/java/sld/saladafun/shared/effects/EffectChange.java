package sld.saladafun.shared.effects;

import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/** One player's add, replace, or remove proposal for an effect type. */
public record EffectChange(
    UUID actorId,
    String typeKey,
    Optional<EffectState> replacement
) {
    public EffectChange {
        Objects.requireNonNull(actorId, "actorId");
        Objects.requireNonNull(typeKey, "typeKey");
        Objects.requireNonNull(replacement, "replacement");
        if (typeKey.isBlank()) {
            throw new IllegalArgumentException("typeKey must not be blank");
        }
        replacement.ifPresent(effect -> {
            if (!typeKey.equals(effect.typeKey())) {
                throw new IllegalArgumentException(
                    "replacement type must match the changed type"
                );
            }
        });
    }

    public static EffectChange replace(UUID actorId, EffectState replacement) {
        return new EffectChange(
            actorId,
            replacement.typeKey(),
            Optional.of(replacement)
        );
    }

    public static EffectChange remove(UUID actorId, String typeKey) {
        return new EffectChange(actorId, typeKey, Optional.empty());
    }
}
