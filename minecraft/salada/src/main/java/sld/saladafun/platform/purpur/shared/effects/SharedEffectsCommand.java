package sld.saladafun.platform.purpur.shared.effects;

import org.bukkit.Server;
import org.bukkit.command.CommandSender;
import org.bukkit.entity.Player;
import sld.saladafun.platform.purpur.shared.SharedCommand;
import sld.saladafun.platform.purpur.shared.SharedModuleCommand;
import sld.saladafun.shared.effects.EffectsState;
import sld.saladafun.shared.effects.SharedEffectsManager;
import sld.saladafun.shared.model.SessionLabel;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.UUID;

/** Administrative command delegate for shared potion effects. */
public final class SharedEffectsCommand implements SharedModuleCommand {
    private static final String PERMISSION = "saladafun.shared.effects.manage";
    private static final List<String> ACTIONS = List.of(
        "enable", "disable", "status", "resume"
    );

    private final Server server;
    private final SharedEffectsManager manager;
    private final PurpurEffectsMapper mapper;
    private final PlayerEffectsSynchronizer synchronizer;
    private final SharedEffectsHandler handler;

    public SharedEffectsCommand(
        Server server,
        SharedEffectsManager manager,
        PurpurEffectsMapper mapper,
        PlayerEffectsSynchronizer synchronizer,
        SharedEffectsHandler handler
    ) {
        this.server = Objects.requireNonNull(server, "server");
        this.manager = Objects.requireNonNull(manager, "manager");
        this.mapper = Objects.requireNonNull(mapper, "mapper");
        this.synchronizer = Objects.requireNonNull(synchronizer, "synchronizer");
        this.handler = Objects.requireNonNull(handler, "handler");
    }

    @Override
    public String moduleName() {
        return "effects";
    }

    @Override
    public boolean execute(CommandSender sender, String[] arguments) {
        if (!sender.hasPermission(PERMISSION)) {
            sender.sendMessage("You do not have permission to manage shared effects.");
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
            sender.sendMessage("Shared effects: " + exception.getMessage());
            return true;
        }
    }

    private boolean enable(CommandSender sender, String[] arguments) {
        if (arguments.length > 2) {
            return usage(sender);
        }
        Map<UUID, EffectsState> backups = onlineSnapshots();
        if (arguments.length == 2) {
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
            "Shared effects enabled as session "
                + manager.activeSession().orElseThrow().label().value() + "."
        );
        return true;
    }

    private boolean disable(CommandSender sender) {
        if (manager.disable().isEmpty()) {
            sender.sendMessage("Shared effects are already disabled.");
            return true;
        }
        for (Player player : server.getOnlinePlayers()) {
            if (player.isDead()) {
                continue;
            }
            manager.pendingRestore(player.getUniqueId()).ifPresent(backup -> {
                synchronizer.restore(player, backup.state());
                manager.markRestored(backup);
            });
        }
        handler.resetReplicas();
        sender.sendMessage("Shared effects disabled; personal effects restored.");
        return true;
    }

    private boolean status(CommandSender sender) {
        if (!manager.isEnabled()) {
            sender.sendMessage("Shared effects are disabled.");
            return true;
        }
        EffectsState state = manager.current().orElseThrow();
        sender.sendMessage(
            "Shared effects are enabled: session "
                + manager.activeSession().orElseThrow().label().value()
                + ", active effects " + state.effects().size()
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
        sender.sendMessage("Resumed shared effects session " + arguments[1] + ".");
        return true;
    }

    private boolean usage(CommandSender sender) {
        sender.sendMessage("Usage: /shared effects <enable|disable|status|resume>");
        return true;
    }

    private Map<UUID, EffectsState> onlineSnapshots() {
        var snapshots = new LinkedHashMap<UUID, EffectsState>();
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
