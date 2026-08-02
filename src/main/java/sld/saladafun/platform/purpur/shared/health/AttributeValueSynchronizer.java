package sld.saladafun.platform.purpur.shared.health;

import org.bukkit.attribute.AttributeInstance;
import org.bukkit.attribute.AttributeModifier;

/** Solves an attribute base value while preserving every installed modifier. */
final class AttributeValueSynchronizer {
    private static final double EPSILON = 1.0E-6;

    void setEffectiveValue(AttributeInstance attribute, double target) {
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
        double base = target / factor - additive;
        attribute.setBaseValue(base);
        if (Math.abs(attribute.getValue() - target) > EPSILON) {
            throw new IllegalStateException(
                "The platform clamped an attempted shared attribute range of " + target
            );
        }
    }
}
