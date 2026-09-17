package sld.saladafun.discordutils.discord;

import java.util.Objects;
import java.util.Optional;

/** Describes a Minecraft player connection transition and an optional disconnect reason. */
public record PlayerPresence(Type type, Optional<String> disconnectReason) {
    /** A completed player login. */
    public static final PlayerPresence JOINED = new PlayerPresence(Type.JOINED, Optional.empty());
    /** A routine player logout with no reason worth displaying. */
    public static final PlayerPresence LEFT = new PlayerPresence(Type.LEFT, Optional.empty());

    /** Validates and normalizes player-presence data. */
    public PlayerPresence {
        Objects.requireNonNull(type, "type");
        Objects.requireNonNull(disconnectReason, "disconnectReason");
        disconnectReason = disconnectReason
            .map(reason -> reason.replaceAll("\\s+", " ").strip())
            .filter(reason -> !reason.isEmpty());
        if (type == Type.JOINED && disconnectReason.isPresent()) {
            throw new IllegalArgumentException("A player login cannot have a disconnect reason");
        }
    }

    /** Creates a player logout carrying a meaningful disconnect reason. */
    public static PlayerPresence disconnected(String reason) {
        return new PlayerPresence(Type.LEFT, Optional.of(Objects.requireNonNull(reason, "reason")));
    }

    /** Identifies the connection transition represented by a presence value. */
    public enum Type {
        /** The player completed login. */
        JOINED,
        /** The player disconnected. */
        LEFT
    }
}
