package sld.saladafun.discordutils.platform.neoforge;

import java.util.List;
import java.util.Objects;
import java.util.function.Consumer;
import java.util.function.Supplier;
import net.minecraft.server.MinecraftServer;
import sld.saladafun.discordutils.discord.OnlinePlayerNamesProvider;

/** Captures online-player names on the dedicated Minecraft server thread. */
public final class MinecraftOnlinePlayerNamesProvider implements OnlinePlayerNamesProvider {
    private final Consumer<Runnable> serverExecutor;
    private final Supplier<List<String>> onlinePlayerNames;

    /** Creates a provider for the currently running dedicated server. */
    public MinecraftOnlinePlayerNamesProvider(MinecraftServer server) {
        MinecraftServer requiredServer = Objects.requireNonNull(server, "server");
        this.serverExecutor = requiredServer::execute;
        this.onlinePlayerNames = () -> requiredServer.getPlayerList().getPlayers().stream()
            .map(player -> player.getGameProfile().getName())
            .toList();
    }

    /** Creates a provider with explicit boundaries for focused scheduling tests. */
    MinecraftOnlinePlayerNamesProvider(
        Consumer<Runnable> serverExecutor,
        Supplier<List<String>> onlinePlayerNames
    ) {
        this.serverExecutor = Objects.requireNonNull(serverExecutor, "serverExecutor");
        this.onlinePlayerNames = Objects.requireNonNull(onlinePlayerNames, "onlinePlayerNames");
    }

    /** Schedules an immutable online-player snapshot on the server thread. */
    @Override
    public void request(Consumer<List<String>> destination) {
        Objects.requireNonNull(destination, "destination");
        serverExecutor.accept(() -> destination.accept(List.copyOf(onlinePlayerNames.get())));
    }
}
