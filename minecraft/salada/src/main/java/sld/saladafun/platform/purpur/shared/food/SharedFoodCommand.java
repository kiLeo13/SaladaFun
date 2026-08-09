package sld.saladafun.platform.purpur.shared.food;

import org.bukkit.Server;
import org.bukkit.command.CommandSender;
import org.bukkit.entity.Player;
import sld.saladafun.platform.purpur.shared.SharedCommand;
import sld.saladafun.platform.purpur.shared.SharedModuleCommand;
import sld.saladafun.shared.food.FoodState;
import sld.saladafun.shared.food.SharedFoodManager;
import sld.saladafun.shared.model.SessionLabel;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.UUID;

/** Administrative command delegate for shared food. */
public final class SharedFoodCommand implements SharedModuleCommand {
    private static final String PERMISSION = "saladafun.shared.food.manage";
    private static final List<String> ACTIONS = List.of(
        "enable", "disable", "status", "resume"
    );

    private final Server server;
    private final SharedFoodManager manager;
    private final PurpurFoodMapper mapper;
    private final PlayerFoodSynchronizer synchronizer;
    private final SharedFoodHandler handler;

    public SharedFoodCommand(
        Server server,
        SharedFoodManager manager,
        PurpurFoodMapper mapper,
        PlayerFoodSynchronizer synchronizer,
        SharedFoodHandler handler
    ) {
        this.server = Objects.requireNonNull(server, "server");
        this.manager = Objects.requireNonNull(manager, "manager");
        this.mapper = Objects.requireNonNull(mapper, "mapper");
        this.synchronizer = Objects.requireNonNull(synchronizer, "synchronizer");
        this.handler = Objects.requireNonNull(handler, "handler");
    }

    @Override
    public String moduleName() {
        return "food";
    }

    @Override
    public boolean execute(CommandSender sender, String[] arguments) {
        if (!sender.hasPermission(PERMISSION)) {
            sender.sendMessage("You do not have permission to manage shared food.");
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
            sender.sendMessage("Shared food: " + exception.getMessage());
            return true;
        }
    }

    private boolean enable(CommandSender sender, String[] arguments) {
        Map<UUID, FoodState> backups = onlineSnapshots();
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
            "Shared food enabled as session "
                + manager.activeSession().orElseThrow().label().value() + "."
        );
        return true;
    }

    private boolean disable(CommandSender sender) {
        if (manager.disable().isEmpty()) {
            sender.sendMessage("Shared food is already disabled.");
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
        sender.sendMessage("Shared food disabled; personal food restored.");
        return true;
    }

    private boolean status(CommandSender sender) {
        if (!manager.isEnabled()) {
            sender.sendMessage("Shared food is disabled.");
            return true;
        }
        FoodState state = manager.current().orElseThrow();
        sender.sendMessage(
            "Shared food is enabled: session "
                + manager.activeSession().orElseThrow().label().value()
                + ", food " + state.foodLevel() + ", saturation "
                + state.saturation() + ", exhaustion " + state.exhaustion()
                + ", revision " + state.revision() + "."
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
        sender.sendMessage("Resumed shared food session " + arguments[1] + ".");
        return true;
    }

    private boolean usage(CommandSender sender) {
        sender.sendMessage("Usage: /shared food <enable|disable|status|resume>");
        return true;
    }

    private Map<UUID, FoodState> onlineSnapshots() {
        var snapshots = new LinkedHashMap<UUID, FoodState>();
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
