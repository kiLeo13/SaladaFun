package sld.saladafun.platform.purpur.shared.effects;

import org.bukkit.NamespacedKey;
import org.bukkit.Registry;
import org.bukkit.entity.Player;
import org.bukkit.potion.PotionEffect;
import org.bukkit.potion.PotionEffectType;
import sld.saladafun.shared.effects.EffectState;
import sld.saladafun.shared.effects.EffectsState;

import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Objects;

/** Maps active Purpur potion effects into portable shared-effects state. */
public final class PurpurEffectsMapper {

    public EffectsState snapshot(Player player, long revision) {
        Objects.requireNonNull(player, "player");
        Map<String, EffectState> effects = new LinkedHashMap<>();
        for (PotionEffect effect : player.getActivePotionEffects()) {
            EffectState state = fromPlatform(effect);
            effects.put(state.typeKey(), state);
        }
        return new EffectsState(effects, revision);
    }

    public void apply(Player player, EffectsState state) {
        Objects.requireNonNull(player, "player");
        Objects.requireNonNull(state, "state");
        for (PotionEffect active : player.getActivePotionEffects()) {
            String type = active.getType().getKey().asString();
            if (!state.effects().containsKey(type)) {
                player.removePotionEffect(active.getType());
            }
        }
        for (EffectState effect : state.effects().values()) {
            PotionEffect active = player.getPotionEffect(type(effect.typeKey()));
            PotionEffect desired = toPlatform(effect);
            if (!desired.equals(active)) {
                player.removePotionEffect(desired.getType());
                player.addPotionEffect(desired);
            }
        }
    }

    private EffectState fromPlatform(PotionEffect effect) {
        return new EffectState(
            effect.getType().getKey().asString(),
            effect.getAmplifier(),
            effect.getDuration(),
            effect.isAmbient(),
            effect.hasParticles(),
            effect.hasIcon(),
            effect.getHiddenPotionEffect() == null
                ? null
                : fromPlatform(effect.getHiddenPotionEffect())
        );
    }

    private PotionEffect toPlatform(EffectState effect) {
        return new PotionEffect(
            type(effect.typeKey()),
            effect.durationTicks(),
            effect.amplifier(),
            effect.ambient(),
            effect.particles(),
            effect.icon(),
            effect.hiddenEffect() == null
                ? null
                : toPlatform(effect.hiddenEffect())
        );
    }

    private PotionEffectType type(String key) {
        NamespacedKey namespacedKey = NamespacedKey.fromString(key);
        PotionEffectType type = namespacedKey == null
            ? null
            : Registry.MOB_EFFECT.get(namespacedKey);
        if (type == null) {
            throw new IllegalArgumentException("Unknown potion effect type: " + key);
        }
        return type;
    }
}
