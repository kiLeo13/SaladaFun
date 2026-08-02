package sld.saladafun.platform.purpur.shared.health;

import org.bukkit.Server;
import org.bukkit.command.CommandSender;
import org.bukkit.entity.Player;
import sld.saladafun.platform.purpur.shared.SharedCommand;
import sld.saladafun.platform.purpur.shared.SharedModuleCommand;
import sld.saladafun.shared.health.HealthState;
import sld.saladafun.shared.health.SharedHealthManager;
import sld.saladafun.shared.model.SessionLabel;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.UUID;

/** Administrative command delegate for shared health. */
public final class SharedHealthCommand implements SharedModuleCommand {
    private static final String PERMISSION = "saladafun.shared.health.manage";
    private static final List<String> ACTIONS = List.of(
        "enable", "disable", "status", "resume"
    );

    private final Server server;
    private final SharedHealthManager manager;
    private final PurpurHealthMapper mapper;
    private final PlayerHealthSynchronizer synchronizer;
    private final SharedHealthHandler handler;

    public SharedHealthCommand(
        Server server,
        SharedHealthManager manager,
        PurpurHealthMapper mapper,
        PlayerHealthSynchronizer synchronizer,
        SharedHealthHandler handler
    ) {
        this.server = Objects.requireNonNull(server, "server");
        this.manager = Objects.requireNonNull(manager, "manager");
        this.mapper = Objects.requireNonNull(mapper, "mapper");
        this.synchronizer = Objects.requireNonNull(synchronizer, "synchronizer");
        this.handler = Objects.requireNonNull(handler, "handler");
    }

    @Override
    public String moduleName() {
        return "health";
    }

    @Override
    public boolean execute(CommandSender sender, String[] arguments) {
        if (!sender.hasPermission(PERMISSION)) {
            sender.sendMessage("You do not have permission to manage shared health.");
            return true;
        }
        try {
            return switch (arguments[0].toLowerCase(Locale.ROOT)) {
                case "enable" -> enable(sender, arguments);
                case "disable" -> disable(sender);
                case "status" -> status(sender);
                case "resume" -> resume(sender, arguments);
                default -> usage(sender);
            };
        } catch (IllegalArgumentException | IllegalStateException exception) {
            sender.sendMessage("Shared health: " + exception.getMessage());
            return true;
        }
    }

    private boolean enable(CommandSender sender, String[] arguments) {
        Map<UUID, HealthState> backups = onlineSnapshots();
        if (arguments.length >= 2) {
            Player source = server.getPlayerExact(arguments[1]);
            if (source == null) {
                sender.sendMessage("The source player must be online.");
                return true;
            }
            manager.enableFrom(
                source.getUniqueId(),
                mapper.snapshot(source, 0),
                backups
            );
        } else {
            manager.enableFresh(backups);
        }
        handler.resetReplicas();
        synchronizer.applyToAll(manager.current().orElseThrow());
        sender.sendMessage(
            "Shared health enabled as session "
                + manager.activeSession().orElseThrow().label().value() + "."
        );
        return true;
    }

    private boolean disable(CommandSender sender) {
        if (manager.disable().isEmpty()) {
            sender.sendMessage("Shared health is already disabled.");
            return true;
        }
        for (Player player : server.getOnlinePlayers()) {
            if (player.isDead()) {
                continue;
            }
            manager.pendingRestore(player.getUniqueId()).ifPresent(backup -> {
                synchronizer.apply(player, backup.state());
                manager.markRestored(backup);
            });
        }
        handler.resetReplicas();
        sender.sendMessage("Shared health disabled; personal health restored.");
        return true;
    }

    private boolean status(CommandSender sender) {
        if (!manager.isEnabled()) {
            sender.sendMessage("Shared health is disabled.");
            return true;
        }
        HealthState state = manager.current().orElseThrow();
        sender.sendMessage(
            "Shared health is enabled: session "
                + manager.activeSession().orElseThrow().label().value()
                + ", health " + state.health() + "/" + state.maximumHealth()
                + ", absorption " + state.absorption() + "/"
                + state.maximumAbsorption() + ", revision " + state.revision() + "."
        );
        return true;
    }

    private boolean resume(CommandSender sender, String[] arguments) {
        if (arguments.length != 2) {
            return usage(sender);
        }
        manager.resume(new SessionLabel(arguments[1]), onlineSnapshots());
        handler.resetReplicas();
        synchronizer.applyToAll(manager.current().orElseThrow());
        sender.sendMessage("Resumed shared health session " + arguments[1] + ".");
        return true;
    }

    private boolean usage(CommandSender sender) {
        sender.sendMessage("Usage: /shared health <enable|disable|status|resume>");
        return true;
    }

    private Map<UUID, HealthState> onlineSnapshots() {
        var snapshots = new LinkedHashMap<UUID, HealthState>();
        for (Player player : server.getOnlinePlayers()) {
            snapshots.put(player.getUniqueId(), mapper.snapshot(player, 0));
        }
        return Map.copyOf(snapshots);
    }

    @Override
    public List<String> complete(CommandSender sender, String[] arguments) {
        if (!sender.hasPermission(PERMISSION)) {
            return List.of();
        }
        if (arguments.length == 1) {
            return SharedCommand.filter(ACTIONS, arguments[0]);
        }
        if (arguments.length == 2 && arguments[0].equalsIgnoreCase("enable")) {
            return SharedCommand.filter(
                server.getOnlinePlayers().stream().map(Player::getName).toList(),
                arguments[1]
            );
        }
        if (arguments.length == 2 && arguments[0].equalsIgnoreCase("resume")) {
            return SharedCommand.filter(
                manager.archivedSessions().stream()
                    .map(session -> session.label().value())
                    .toList(),
                arguments[1]
            );
        }
        return List.of();
    }
}
