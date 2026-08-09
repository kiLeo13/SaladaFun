package sld.saladafun.shared.effects;

import java.util.Comparator;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;

/** Thread-safe canonical effect aggregate with per-type deterministic LWW. */
public final class SharedEffects {
    private EffectsState state;

    public SharedEffects(EffectsState initialState) {
        state = Objects.requireNonNull(initialState, "initialState");
    }

    public synchronized EffectsState snapshot() {
        return state;
    }

    public synchronized EffectsState applyTick(List<EffectChange> changes) {
        Objects.requireNonNull(changes, "changes");
        Map<String, EffectChange> winners = new HashMap<>();
        Comparator<EffectChange> actorOrder = Comparator.comparing(
            change -> change.actorId().toString()
        );
        for (EffectChange change : changes) {
            winners.merge(
                change.typeKey(),
                change,
                (left, right) -> actorOrder.compare(left, right) >= 0 ? left : right
            );
        }
        if (winners.isEmpty()) {
            return state;
        }
        Map<String, EffectState> next = new HashMap<>(state.effects());
        winners.forEach((type, change) -> {
            if (change.replacement().isPresent()) {
                next.put(type, change.replacement().orElseThrow());
            } else {
                next.remove(type);
            }
        });
        Map<String, EffectState> immutable = Map.copyOf(next);
        if (!immutable.equals(state.effects())) {
            state = new EffectsState(immutable, state.revision() + 1);
        }
        return state;
    }

    /** Refreshes remaining durations without creating a gameplay revision. */
    public synchronized EffectsState refreshDurations(
        Map<String, EffectState> observed
    ) {
        Objects.requireNonNull(observed, "observed");
        Map<String, EffectState> refreshed = new HashMap<>(state.effects());
        state.effects().forEach((type, canonical) -> {
            EffectState current = observed.get(type);
            if (canonical.sameDefinition(current)) {
                refreshed.put(type, current);
            }
        });
        state = new EffectsState(refreshed, state.revision());
        return state;
    }
}
