package sld.saladafun.platform.purpur.shared;

import org.bukkit.Server;
import org.bukkit.command.Command;
import org.bukkit.command.CommandExecutor;
import org.bukkit.command.CommandSender;
import org.bukkit.command.TabCompleter;
import org.bukkit.entity.Player;
import sld.saladafun.shared.inventory.SharedInventoryManager;
import sld.saladafun.shared.inventory.model.InventorySnapshot;
import sld.saladafun.shared.inventory.model.SessionLabel;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.UUID;

/**
 * Administrative command for shared-inventory lifecycle operations.
 */
public final class SharedInventoryCommand implements CommandExecutor, TabCompleter {
    private final Server server;
    private final SharedInventoryManager manager;
    private final PurpurInventoryMapper mapper;
    private final PlayerInventorySynchronizer synchronizer;

    public SharedInventoryCommand(
        Server server,
        SharedInventoryManager manager,
        PurpurInventoryMapper mapper,
        PlayerInventorySynchronizer synchronizer
    ) {
        this.server = Objects.requireNonNull(server, "server");
        this.manager = Objects.requireNonNull(manager, "manager");
        this.mapper = Objects.requireNonNull(mapper, "mapper");
        this.synchronizer = Objects.requireNonNull(synchronizer, "synchronizer");
    }

    @Override
    public boolean onCommand(
        CommandSender sender,
        Command command,
        String label,
        String[] arguments
    ) {
        if (arguments.length < 2 || !arguments[0].equalsIgnoreCase("inventory")) {
            sender.sendMessage("Usage: /shared inventory <enable|disable|status|resume>");
            return true;
        }
        try {
            return switch (arguments[1].toLowerCase(Locale.ROOT)) {
                case "enable" -> enable(sender, arguments);
                case "disable" -> disable(sender);
                case "status" -> status(sender);
                case "resume" -> resume(sender, arguments);
                default -> {
                    sender.sendMessage(
                        "Usage: /shared inventory <enable|disable|status|resume>"
                    );
                    yield true;
                }
            };
        } catch (IllegalArgumentException | IllegalStateException exception) {
            sender.sendMessage("Shared inventory: " + exception.getMessage());
            return true;
        }
    }

    private boolean enable(CommandSender sender, String[] arguments) {
        closeInventories();
        Map<UUID, InventorySnapshot> backups = onlineSnapshots();
        if (arguments.length >= 3) {
            Player source = server.getPlayerExact(arguments[2]);
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
            manager.enableEmpty(backups);
        }
        InventorySnapshot canonical = manager.current().orElseThrow();
        synchronizer.applyToAll(canonical);
        sender.sendMessage(
            "Shared inventory enabled as session "
                + manager.activeSession().orElseThrow().label().value() + "."
        );
        return true;
    }

    private boolean disable(CommandSender sender) {
        if (manager.disable().isEmpty()) {
            sender.sendMessage("Shared inventory is already disabled.");
            return true;
        }
        for (Player player : server.getOnlinePlayers()) {
            manager.pendingRestore(player.getUniqueId()).ifPresent(backup -> {
                synchronizer.apply(player, backup.inventory());
                manager.markRestored(backup);
            });
        }
        sender.sendMessage("Shared inventory disabled; personal inventories restored.");
        return true;
    }

    private boolean status(CommandSender sender) {
        if (!manager.isEnabled()) {
            sender.sendMessage("Shared inventory is disabled.");
            return true;
        }
        var session = manager.activeSession().orElseThrow();
        sender.sendMessage(
            "Shared inventory is enabled: session " + session.label().value()
                + ", revision " + manager.current().orElseThrow().revision() + "."
        );
        return true;
    }

    private boolean resume(CommandSender sender, String[] arguments) {
        if (arguments.length != 3) {
            sender.sendMessage("Usage: /shared inventory resume <session>");
            return true;
        }
        closeInventories();
        manager.resume(new SessionLabel(arguments[2]), onlineSnapshots());
        synchronizer.applyToAll(manager.current().orElseThrow());
        sender.sendMessage("Resumed shared session " + arguments[2] + ".");
        return true;
    }

    private void closeInventories() {
        for (Player player : server.getOnlinePlayers()) {
            player.closeInventory();
        }
    }

    private Map<UUID, InventorySnapshot> onlineSnapshots() {
        var snapshots = new LinkedHashMap<UUID, InventorySnapshot>();
        for (Player player : server.getOnlinePlayers()) {
            snapshots.put(player.getUniqueId(), mapper.snapshot(player, 0));
        }
        return Map.copyOf(snapshots);
    }

    @Override
    public List<String> onTabComplete(
        CommandSender sender,
        Command command,
        String alias,
        String[] arguments
    ) {
        if (arguments.length == 1) {
            return filter(List.of("inventory"), arguments[0]);
        }
        if (!arguments[0].equalsIgnoreCase("inventory")) {
            return List.of();
        }
        if (arguments.length == 2) {
            return filter(
                List.of("enable", "disable", "status", "resume"),
                arguments[1]
            );
        }
        if (arguments.length == 3 && arguments[1].equalsIgnoreCase("enable")) {
            return filter(
                server.getOnlinePlayers().stream().map(Player::getName).toList(),
                arguments[2]
            );
        }
        if (arguments.length == 3 && arguments[1].equalsIgnoreCase("resume")) {
            return filter(
                manager.archivedSessions().stream()
                    .map(session -> session.label().value())
                    .toList(),
                arguments[2]
            );
        }
        return List.of();
    }

    private List<String> filter(List<String> values, String prefix) {
        String normalized = prefix.toLowerCase(Locale.ROOT);
        return values.stream()
            .filter(value -> value.toLowerCase(Locale.ROOT).startsWith(normalized))
            .toList();
    }
}
