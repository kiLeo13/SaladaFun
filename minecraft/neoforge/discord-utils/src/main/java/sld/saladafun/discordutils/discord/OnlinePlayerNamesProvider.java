package sld.saladafun.discordutils.discord;

import java.util.List;
import java.util.function.Consumer;

/** Supplies immutable online-player name snapshots without exposing Minecraft types. */
@FunctionalInterface
public interface OnlinePlayerNamesProvider {
    /** Requests a snapshot and passes it to the destination on the provider's execution context. */
    void request(Consumer<List<String>> destination);
}
