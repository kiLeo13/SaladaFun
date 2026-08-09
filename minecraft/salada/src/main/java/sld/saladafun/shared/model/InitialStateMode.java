package sld.saladafun.shared.model;

/** Defines how a shared-state session obtained its initial canonical value. */
public enum InitialStateMode {
    FRESH,
    SOURCE_PLAYER,
    RESUMED
}
