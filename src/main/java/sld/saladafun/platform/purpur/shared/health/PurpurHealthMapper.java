package sld.saladafun.platform.purpur.shared.health;

import org.bukkit.attribute.Attribute;
import org.bukkit.attribute.AttributeInstance;
import org.bukkit.entity.Player;
import sld.saladafun.shared.health.HealthPhase;
import sld.saladafun.shared.health.HealthState;

import java.util.Objects;

/** Maps Purpur player health and attribute ranges into portable domain state. */
public final class PurpurHealthMapper {
    private final AttributeValueSynchronizer attributeSynchronizer =
        new AttributeValueSynchronizer();

    public HealthState snapshot(Player player, long revision) {
        Objects.requireNonNull(player, "player");
        double maximumHealth = attribute(player, Attribute.MAX_HEALTH).getValue();
        double maximumAbsorption = attribute(player, Attribute.MAX_ABSORPTION).getValue();
        double health = Math.min(player.getHealth(), maximumHealth);
        double absorption = Math.min(player.getAbsorptionAmount(), maximumAbsorption);
        HealthPhase phase = player.isDead() || health == 0.0
            ? HealthPhase.DEAD
            : HealthPhase.ALIVE;
        if (phase == HealthPhase.DEAD) {
            health = 0.0;
            absorption = 0.0;
        }
        return new HealthState(
            health,
            maximumHealth,
            absorption,
            maximumAbsorption,
            phase,
            revision
        );
    }

    public void apply(Player player, HealthState state) {
        Objects.requireNonNull(player, "player");
        Objects.requireNonNull(state, "state");
        attributeSynchronizer.setEffectiveValue(
            attribute(player, Attribute.MAX_HEALTH),
            state.maximumHealth()
        );
        attributeSynchronizer.setEffectiveValue(
            attribute(player, Attribute.MAX_ABSORPTION),
            state.maximumAbsorption()
        );
        if (player.isDead()) {
            return;
        }
        player.setAbsorptionAmount(state.absorption());
        player.setHealth(state.health());
    }

    /** Removes SaladaFun range overrides and restores personal current values. */
    public void restore(Player player, HealthState personalState) {
        Objects.requireNonNull(player, "player");
        Objects.requireNonNull(personalState, "personalState");
        AttributeInstance health = attribute(player, Attribute.MAX_HEALTH);
        AttributeInstance absorption = attribute(player, Attribute.MAX_ABSORPTION);
        attributeSynchronizer.clear(health);
        attributeSynchronizer.clear(absorption);
        if (player.isDead()) {
            return;
        }
        player.setAbsorptionAmount(Math.min(
            personalState.absorption(), absorption.getValue()
        ));
        player.setHealth(Math.min(personalState.health(), health.getValue()));
    }

    public double naturalMaximumHealth(Player player) {
        return attributeSynchronizer.naturalValue(
            attribute(player, Attribute.MAX_HEALTH)
        );
    }

    public double naturalMaximumAbsorption(Player player) {
        return attributeSynchronizer.naturalValue(
            attribute(player, Attribute.MAX_ABSORPTION)
        );
    }

    private AttributeInstance attribute(Player player, Attribute attribute) {
        AttributeInstance instance = player.getAttribute(attribute);
        if (instance == null) {
            throw new IllegalStateException("Player is missing attribute " + attribute);
        }
        return instance;
    }
}
