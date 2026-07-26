package sld.saladafun.platform.purpur.batch;

import org.bukkit.Bukkit;
import org.bukkit.Chunk;
import org.bukkit.ChunkSnapshot;
import org.bukkit.Material;
import org.bukkit.World;
import org.bukkit.block.Block;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.block.BlockBreakEvent;
import org.bukkit.plugin.java.JavaPlugin;
import org.bukkit.scheduler.BukkitTask;
import sld.saladafun.batchbreaking.BatchBlockAction;
import sld.saladafun.batchbreaking.BatchBreakingSetting;
import sld.saladafun.batchbreaking.BlockPosition;
import sld.saladafun.batchbreaking.CubicRange;
import sld.saladafun.batchbreaking.ToolDurabilityMode;
import sld.saladafun.platform.purpur.config.PluginSettings;

import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.IdentityHashMap;
import java.util.Map;
import java.util.Objects;
import java.util.UUID;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.atomic.AtomicInteger;

/**
 * Streams chunk-snapshot scans off-thread and performs bounded player-aware breaks on-thread.
 */
public final class BatchBreakingHandler implements Listener, AutoCloseable {
    private final JavaPlugin plugin;
    private final PluginSettings settings;
    private final CubicRange range = new CubicRange();
    private final BatchBlockExecutor blockExecutor = new BatchBlockExecutor();
    private final ExecutorService scanner = Executors.newSingleThreadExecutor(
        runnable -> Thread.ofPlatform().name("saladafun-batch-scanner").daemon(true)
            .unstarted(runnable)
    );
    private final Map<BlockBreakEvent, Trigger> pendingTriggers = new IdentityHashMap<>();
    private final Map<UUID, BatchJob> activeJobs = new HashMap<>();
    private final Map<UUID, BatchBlockAction> generatedBreaks = new HashMap<>();

    public BatchBreakingHandler(JavaPlugin plugin, PluginSettings settings) {
        this.plugin = Objects.requireNonNull(plugin, "plugin");
        this.settings = Objects.requireNonNull(settings, "settings");
    }

    @EventHandler(priority = EventPriority.LOWEST, ignoreCancelled = true)
    public void captureTrigger(BlockBreakEvent event) {
        Player player = event.getPlayer();
        BatchBlockAction generatedAction = generatedBreaks.get(player.getUniqueId());
        if (generatedAction != null) {
            BatchBreakEventPolicy.apply(generatedAction, event);
            return;
        }
        if (!player.hasPermission("saladafun.batchbreaking.use")) {
            return;
        }
        BatchBreakingSetting setting = settings.batchBreakingSetting();
        if (setting instanceof BatchBreakingSetting.Disabled) {
            return;
        }
        Block block = event.getBlock();
        pendingTriggers.put(
            event,
            new Trigger(
                player,
                block.getWorld(),
                block.getType(),
                new BlockPosition(block.getX(), block.getY(), block.getZ()),
                setting,
                settings.batchBlockAction(),
                settings.toolDurabilityMode(),
                block.getWorld().getMinHeight(),
                block.getWorld().getMaxHeight()
            )
        );
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void startAfterFinalOutcome(BlockBreakEvent event) {
        Trigger trigger = pendingTriggers.remove(event);
        if (trigger == null || event.isCancelled()) {
            return;
        }
        if (activeJobs.containsKey(trigger.player().getUniqueId())) {
            return;
        }
        BatchJob job = new BatchJob(trigger);
        activeJobs.put(trigger.player().getUniqueId(), job);
        job.start();
    }

    @Override
    public void close() {
        for (BatchJob job : new ArrayList<>(activeJobs.values())) {
            job.cancel();
        }
        scanner.shutdownNow();
    }

    private final class BatchJob {
        private final Trigger trigger;
        private final ArrayDeque<Chunk> chunks;
        private final ArrayBlockingQueue<BlockPosition> matches;
        private final AtomicInteger scansInFlight = new AtomicInteger();
        private final ArrayList<Future<?>> futures = new ArrayList<>();
        private BukkitTask captureTask;
        private BukkitTask breakTask;
        private boolean captureComplete;
        private volatile boolean cancelled;
        private long broken;

        private BatchJob(Trigger trigger) {
            this.trigger = trigger;
            chunks = new ArrayDeque<>(
                trigger.player().getSentChunks().stream()
                    .filter(chunk -> chunk.getWorld().equals(trigger.world()))
                    .toList()
            );
            matches = new ArrayBlockingQueue<>(settings.maxQueuedMatches());
        }

        private void start() {
            captureTask = Bukkit.getScheduler().runTaskTimer(
                plugin, this::captureSnapshots, 1L, 1L
            );
            breakTask = Bukkit.getScheduler().runTaskTimer(
                plugin, this::breakMatches, 1L, 1L
            );
        }

        private void captureSnapshots() {
            if (!isPlayerValid()) {
                cancel();
                return;
            }
            int budget = settings.snapshotChunksPerTick();
            while (budget-- > 0
                && scansInFlight.get() < settings.snapshotChunksPerTick()) {
                Chunk chunk = chunks.pollFirst();
                if (chunk == null) {
                    captureComplete = true;
                    captureTask.cancel();
                    return;
                }
                if (!trigger.player().getSentChunks().contains(chunk)
                    || !trigger.world().isChunkLoaded(chunk.getX(), chunk.getZ())) {
                    continue;
                }
                ChunkSnapshot snapshot = chunk.getChunkSnapshot(false, false, false, false);
                scansInFlight.incrementAndGet();
                futures.add(scanner.submit(() -> scan(snapshot)));
            }
        }

        private void scan(ChunkSnapshot snapshot) {
            try {
                int baseX = snapshot.getX() << 4;
                int baseZ = snapshot.getZ() << 4;
                for (int y = trigger.minimumY();
                     y < trigger.maximumY() && !cancelled;
                     y++) {
                    for (int localX = 0; localX < 16; localX++) {
                        for (int localZ = 0; localZ < 16; localZ++) {
                            if (snapshot.getBlockType(localX, y, localZ) != trigger.material()) {
                                continue;
                            }
                            BlockPosition position = new BlockPosition(
                                baseX + localX, y, baseZ + localZ
                            );
                            if (position.equals(trigger.origin())
                                || !range.contains(
                                    trigger.origin(), position, trigger.setting()
                                )) {
                                continue;
                            }
                            matches.put(position);
                        }
                    }
                }
            } catch (InterruptedException interrupted) {
                Thread.currentThread().interrupt();
            } finally {
                scansInFlight.decrementAndGet();
            }
        }

        private void breakMatches() {
            if (!isPlayerValid()) {
                cancel();
                return;
            }
            int budget = settings.blocksPerTick();
            while (budget-- > 0) {
                BlockPosition position = matches.poll();
                if (position == null) {
                    break;
                }
                if (!trigger.world().isChunkLoaded(position.chunkX(), position.chunkZ())) {
                    continue;
                }
                Chunk chunk = trigger.world().getChunkAt(position.chunkX(), position.chunkZ());
                if (!trigger.player().getSentChunks().contains(chunk)) {
                    continue;
                }
                Block block = trigger.world().getBlockAt(
                    position.x(), position.y(), position.z()
                );
                if (block.getType() != trigger.material()) {
                    continue;
                }
                UUID playerId = trigger.player().getUniqueId();
                generatedBreaks.put(playerId, trigger.action());
                try {
                    if (blockExecutor.breakBlock(
                        trigger.player(),
                        block,
                        trigger.action(),
                        trigger.durabilityMode()
                    )) {
                        broken++;
                    }
                } finally {
                    generatedBreaks.remove(playerId);
                }
            }
            if (captureComplete && scansInFlight.get() == 0 && matches.isEmpty()) {
                finish();
            }
        }

        private boolean isPlayerValid() {
            return !cancelled
                && trigger.player().isOnline()
                && trigger.player().getWorld().equals(trigger.world());
        }

        private void finish() {
            cancelTasks();
            activeJobs.remove(trigger.player().getUniqueId());
            trigger.player().sendMessage("Batch breaking completed: " + broken + " blocks.");
        }

        private void cancel() {
            cancelled = true;
            cancelTasks();
            for (Future<?> future : futures) {
                future.cancel(true);
            }
            activeJobs.remove(trigger.player().getUniqueId());
        }

        private void cancelTasks() {
            if (captureTask != null && !captureTask.isCancelled()) {
                captureTask.cancel();
            }
            if (breakTask != null && !breakTask.isCancelled()) {
                breakTask.cancel();
            }
        }
    }

    private record Trigger(
        Player player,
        World world,
        Material material,
        BlockPosition origin,
        BatchBreakingSetting setting,
        BatchBlockAction action,
        ToolDurabilityMode durabilityMode,
        int minimumY,
        int maximumY
    ) {
    }
}
