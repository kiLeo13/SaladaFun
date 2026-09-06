package sld.saladafun.platform.purpur.shared.health;

import org.bukkit.NamespacedKey;
import org.bukkit.attribute.AttributeInstance;
import org.bukkit.attribute.AttributeModifier;

/** Applies a transient effective-value override while preserving external modifiers. */
final class AttributeValueSynchronizer {
    private static final double EPSILON = 1.0E-6;
    private static final NamespacedKey OVERRIDE_KEY = new NamespacedKey(
        "saladafun", "shared_range_override"
    );

    void setEffectiveValue(AttributeInstance attribute, double target) {
        if (Math.abs(attribute.getValue() - target) <= EPSILON) {
            return;
        }
        attribute.removeModifier(OVERRIDE_KEY);
        double additive = 0.0;
        double scalar = 0.0;
        double multiplier = 1.0;
        for (AttributeModifier modifier : attribute.getModifiers()) {
            switch (modifier.getOperation()) {
                case ADD_NUMBER -> additive += modifier.getAmount();
                case ADD_SCALAR -> scalar += modifier.getAmount();
                case MULTIPLY_SCALAR_1 -> multiplier *= 1.0 + modifier.getAmount();
            }
        }
        double factor = (1.0 + scalar) * multiplier;
        if (!Double.isFinite(factor) || Math.abs(factor) < EPSILON) {
            throw new IllegalStateException(
                "Cannot synchronize an attribute whose modifiers collapse its range"
            );
        }
        double override = target / factor - attribute.getBaseValue() - additive;
        attribute.addTransientModifier(new AttributeModifier(
            OVERRIDE_KEY,
            override,
            AttributeModifier.Operation.ADD_NUMBER
        ));
        if (Math.abs(attribute.getValue() - target) > EPSILON) {
            throw new IllegalStateException(
                "The platform clamped an attempted shared attribute range of " + target
            );
        }
    }

    void clear(AttributeInstance attribute) {
        attribute.removeModifier(OVERRIDE_KEY);
    }

    double naturalValue(AttributeInstance attribute) {
        double additive = 0.0;
        double scalar = 0.0;
        double multiplier = 1.0;
        for (AttributeModifier modifier : attribute.getModifiers()) {
            if (OVERRIDE_KEY.equals(modifier.getKey())) {
                continue;
            }
            switch (modifier.getOperation()) {
                case ADD_NUMBER -> additive += modifier.getAmount();
                case ADD_SCALAR -> scalar += modifier.getAmount();
                case MULTIPLY_SCALAR_1 -> multiplier *= 1.0 + modifier.getAmount();
            }
        }
        return (attribute.getBaseValue() + additive) * (1.0 + scalar) * multiplier;
    }
}
