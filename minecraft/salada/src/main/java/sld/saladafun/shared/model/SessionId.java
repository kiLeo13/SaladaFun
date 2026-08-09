package sld.saladafun.shared.model;

import java.util.Objects;
import java.util.UUID;

/** Stable database identity for a shared-state session. */
public record SessionId(UUID value) {
    public SessionId {
        Objects.requireNonNull(value, "value");
    }

    public static SessionId create() {
        return new SessionId(UUID.randomUUID());
    }
}
