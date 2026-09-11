package sld.saladafun.discordutils.discord;

import java.util.Objects;
import java.util.function.BooleanSupplier;
import java.util.function.Consumer;
import java.util.function.Supplier;

/** Executes the online-players command across the Minecraft-to-Discord boundary. */
final class DiscordPlayersCommandHandler implements Consumer<DiscordMessageCommandRequest> {
    private final OnlinePlayerNamesProvider onlinePlayerNames;
    private final BooleanSupplier active;
    private final Supplier<DiscordPlayersCommandResponder> responder;

    /** Creates a handler whose delayed responses remain bound to one Discord session. */
    DiscordPlayersCommandHandler(
        OnlinePlayerNamesProvider onlinePlayerNames,
        BooleanSupplier active,
        Supplier<DiscordPlayersCommandResponder> responder
    ) {
        this.onlinePlayerNames = Objects.requireNonNull(onlinePlayerNames, "onlinePlayerNames");
        this.active = Objects.requireNonNull(active, "active");
        this.responder = Objects.requireNonNull(responder, "responder");
    }

    /** Requests an online-player snapshot for one consumed command. */
    @Override
    public void accept(DiscordMessageCommandRequest request) {
        Objects.requireNonNull(request, "request");
        onlinePlayerNames.request(playerNames -> {
            DiscordPlayersCommandResponder currentResponder = responder.get();
            if (active.getAsBoolean() && currentResponder != null) {
                currentResponder.send(playerNames);
            }
        });
    }
}
