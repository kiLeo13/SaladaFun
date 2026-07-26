package sld.saladafun.platform.purpur.batch;

import org.bukkit.Bukkit;
import org.bukkit.Chunk;
import org.bukkit.ChunkSnapshot;
import org.bukkit.Material;
import org.bukkit.World;
import org.bukkit.block.Block;
import org.bukkit.entity.Animals;
import org.bukkit.entity.Entity;
import org.bukkit.entity.EntityType;
import org.bukkit.entity.Player;
import org.bukkit.entity.Projectile;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.block.BlockBreakEvent;
import org.bukkit.event.block.BlockDropItemEvent;
import org.bukkit.event.entity.EntityDamageByEntityEvent;
import org.bukkit.event.player.PlayerBucketFillEvent;
import org.bukkit.plugin.java.JavaPlugin;
import org.bukkit.projectiles.ProjectileSource;
import org.bukkit.scheduler.BukkitTask;
import sld.saladafun.batchbreaking.BatchBlockAction;
import sld.saladafun.batchbreaking.BatchBreakingSetting;
import sld.saladafun.batchbreaking.BatchExecutionMode;
import sld.saladafun.batchbreaking.BatchSearchSpace;
import sld.saladafun.batchbreaking.BlockPosition;
import sld.saladafun.batchbreaking.ToolDurabilityMode;
import sld.saladafun.platform.purpur.config.PluginSettings;

import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashMap;
import java.util.HashSet;
import java.util.IdentityHashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.atomic.AtomicInteger;

/**
 * Finds matching blocks off-thread when configured and applies all world mutations on-thread.
 */
public final class BatchBreakingHandler implements Listener, AutoCloseable {
    private static final int ASYNC_SNAPSHOTS_PER_TICK = 8;
    private static final int ASYNC_CHANGES_PER_TICK = 512;
    private static final int ASYNC_MATCH_QUEUE_CAPACITY = 16_384;
    private static final double GENERATED_ANIMAL_DAMAGE = 1_000_000.0;

    private final JavaPlugin plugin;
    private final PluginSettings settings;
    private final BatchBlockExecutor blockExecutor = new BatchBlockExecutor();
    private final ExecutorService scanner = Executors.newSingleThreadExecutor(
        runnable -> Thread.ofPlatform().name("saladafun-batch-scanner").daemon(true)
            .unstarted(runnable)
    );
    private final Map<BlockBreakEvent, BlockTrigger> pendingBlockTriggers =
        new IdentityHashMap<>();
    private final Map<PlayerBucketFillEvent, BlockTrigger> pendingBucketTriggers =
        new IdentityHashMap<>();
    private final Map<EntityDamageByEntityEvent, AnimalTrigger> pendingAnimalTriggers =
        new IdentityHashMap<>();
    private final Map<UUID, ActiveJob> activeJobs = new HashMap<>();
    private final Map<UUID, BatchBlockAction> generatedBreaks = new HashMap<>();
    private final Set<UUID> generatedAnimalDamage = new HashSet<>();

    public BatchBreakingHandler(JavaPlugin plugin, PluginSettings settings) {
        this.plugin = Objects.requireNonNull(plugin, "plugin");
        this.settings = Objects.requireNonNull(settings, "settings");
    }

    @EventHandler(priority = EventPriority.LOWEST, ignoreCancelled = true)
    public void captureBlockTrigger(BlockBreakEvent event) {
        Player player = event.getPlayer();
        BatchBlockAction generatedAction = generatedBreaks.get(player.getUniqueId());
        if (generatedAction != null) {
            BatchBreakEventPolicy.apply(generatedAction, event);
            return;
        }
        if (!canStartFor(player)) {
            return;
        }
        Block block = event.getBlock();
        pendingBlockTriggers.put(event, blockTrigger(player, block, false));
    }

    @EventHandler(priority = EventPriority.HIGHEST, ignoreCancelled = true)
    public void enforceGeneratedBreakPolicy(BlockBreakEvent event) {
        BatchBlockAction action = generatedBreaks.get(event.getPlayer().getUniqueId());
        if (action != null) {
            BatchBreakEventPolicy.apply(action, event);
        }
    }

    @EventHandler(priority = EventPriority.HIGHEST, ignoreCancelled = true)
    public void suppressGeneratedDrops(BlockDropItemEvent event) {
        BatchBlockAction action = generatedBreaks.get(event.getPlayer().getUniqueId());
        if (action != null) {
            BatchBreakEventPolicy.apply(action, event);
        }
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void startAfterBlockOutcome(BlockBreakEvent event) {
        BlockTrigger trigger = pendingBlockTriggers.remove(event);
        if (trigger != null && !event.isCancelled()) {
            startBlockJob(trigger);
        }
    }

    @EventHandler(priority = EventPriority.LOWEST, ignoreCancelled = true)
    public void captureBucketTrigger(PlayerBucketFillEvent event) {
        Player player = event.getPlayer();
        Block fluid = event.getBlock();
        if (!canStartFor(player) || fluid.getType() != Material.WATER) {
            return;
        }
        pendingBucketTriggers.put(event, blockTrigger(player, fluid, true));
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void startAfterBucketOutcome(PlayerBucketFillEvent event) {
        BlockTrigger trigger = pendingBucketTriggers.remove(event);
        if (trigger != null && !event.isCancelled()) {
            startBlockJob(trigger);
        }
    }

    @EventHandler(priority = EventPriority.LOWEST, ignoreCancelled = true)
    public void captureAnimalTrigger(EntityDamageByEntityEvent event) {
        if (!settings.includeAnimals() || !(event.getEntity() instanceof Animals animal)) {
            return;
        }
        Player player = attackingPlayer(event.getDamager());
        if (player == null
            || generatedAnimalDamage.contains(player.getUniqueId())
            || !canStartFor(player)) {
            return;
        }
        pendingAnimalTriggers.put(
            event,
            new AnimalTrigger(
                player,
                animal.getWorld(),
                animal.getType(),
                animal.getUniqueId(),
                blockPosition(animal),
                settings.batchBreakingSetting(),
                animal
            )
        );
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void startAfterAnimalOutcome(EntityDamageByEntityEvent event) {
        AnimalTrigger trigger = pendingAnimalTriggers.remove(event);
        if (trigger == null
            || event.isCancelled()
            || event.getFinalDamage() < trigger.original().getHealth()) {
            return;
        }
        startAnimalJob(trigger);
    }

    @Override
    public void close() {
        for (ActiveJob job : new ArrayList<>(activeJobs.values())) {
            job.cancel();
        }
        pendingBlockTriggers.clear();
        pendingBucketTriggers.clear();
        pendingAnimalTriggers.clear();
        scanner.shutdownNow();
    }

    private boolean canStartFor(Player player) {
        return player.hasPermission("saladafun.batchbreaking.use")
            && !(settings.batchBreakingSetting() instanceof BatchBreakingSetting.Disabled);
    }

    private BlockTrigger blockTrigger(Player player, Block block, boolean fluid) {
        return new BlockTrigger(
            player,
            block.getWorld(),
            block.getType(),
            new BlockPosition(block.getX(), block.getY(), block.getZ()),
            settings.batchBreakingSetting(),
            settings.batchBlockAction(),
            settings.toolDurabilityMode(),
            block.getWorld().getMinHeight(),
            block.getWorld().getMaxHeight(),
            fluid
        );
    }

    private void startBlockJob(BlockTrigger trigger) {
        if (activeJobs.containsKey(trigger.player().getUniqueId())) {
            return;
        }
        ActiveJob job = settings.batchExecutionMode() == BatchExecutionMode.SYNC
            ? new SynchronousBlockJob(trigger)
            : new AsynchronousBlockJob(trigger);
        startJob(trigger.player(), job);
    }

    private void startAnimalJob(AnimalTrigger trigger) {
        if (activeJobs.containsKey(trigger.player().getUniqueId())) {
            return;
        }
        List<Animals> targets = animalTargets(trigger);
        ActiveJob job = settings.batchExecutionMode() == BatchExecutionMode.SYNC
            ? new SynchronousAnimalJob(trigger.player(), targets)
            : new AsynchronousAnimalJob(trigger.player(), targets);
        startJob(trigger.player(), job);
    }

    private void startJob(Player player, ActiveJob job) {
        activeJobs.put(player.getUniqueId(), job);
        try {
            job.start();
        } catch (RuntimeException exception) {
            activeJobs.remove(player.getUniqueId(), job);
            job.cancel();
            throw exception;
        }
    }

    private List<Chunk> relevantChunks(
        Player player,
        World world,
        BatchSearchSpace searchSpace,
        BlockPosition origin
    ) {
        return player.getSentChunks().stream()
            .filter(chunk -> chunk.getWorld().equals(world))
            .filter(chunk -> world.isChunkLoaded(chunk.getX(), chunk.getZ()))
            .filter(chunk -> searchSpace.intersectsChunk(chunk.getX(), chunk.getZ()))
            .sorted(Comparator.comparingLong(chunk -> chunkDistanceSquared(chunk, origin)))
            .toList();
    }

    private long chunkDistanceSquared(Chunk chunk, BlockPosition origin) {
        long dx = (long) chunk.getX() - origin.chunkX();
        long dz = (long) chunk.getZ() - origin.chunkZ();
        return dx * dx + dz * dz;
    }

    private void scanSnapshot(
        ChunkSnapshot snapshot,
        BlockTrigger trigger,
        BatchSearchSpace searchSpace,
        PositionSink sink
    ) throws InterruptedException {
        int chunkX = snapshot.getX();
        int chunkZ = snapshot.getZ();
        int baseX = chunkX << 4;
        int baseZ = chunkZ << 4;
        int minimumLocalX = searchSpace.minimumLocalX(chunkX);
        int maximumLocalX = searchSpace.maximumLocalXExclusive(chunkX);
        int minimumLocalZ = searchSpace.minimumLocalZ(chunkZ);
        int maximumLocalZ = searchSpace.maximumLocalZExclusive(chunkZ);

        for (int y = searchSpace.minimumY();
             y < searchSpace.maximumYExclusive();
             y++) {
            if (Thread.currentThread().isInterrupted()) {
                throw new InterruptedException();
            }
            for (int localX = minimumLocalX; localX < maximumLocalX; localX++) {
                for (int localZ = minimumLocalZ; localZ < maximumLocalZ; localZ++) {
                    if (snapshot.getBlockType(localX, y, localZ) != trigger.material()) {
                        continue;
                    }
                    BlockPosition position = new BlockPosition(
                        baseX + localX, y, baseZ + localZ
                    );
                    if (!position.equals(trigger.origin())) {
                        sink.accept(position);
                    }
                }
            }
        }
    }

    private boolean applyBlock(BlockTrigger trigger, BlockPosition position) {
        if (!trigger.world().isChunkLoaded(position.chunkX(), position.chunkZ())) {
            return false;
        }
        Chunk chunk = trigger.world().getChunkAt(position.chunkX(), position.chunkZ());
        if (!trigger.player().isChunkSent(chunk)) {
            return false;
        }
        Block block = trigger.world().getBlockAt(position.x(), position.y(), position.z());
        if (block.getType() != trigger.material()) {
            return false;
        }

        UUID playerId = trigger.player().getUniqueId();
        BatchBlockAction action = trigger.fluid()
            ? BatchBlockAction.NO_DROPS
            : trigger.action();
        generatedBreaks.put(playerId, action);
        try {
            if (trigger.fluid()) {
                BlockBreakEvent event = new BlockBreakEvent(block, trigger.player());
                event.setDropItems(false);
                event.setExpToDrop(0);
                Bukkit.getPluginManager().callEvent(event);
                if (event.isCancelled()) {
                    return false;
                }
                block.setType(Material.AIR, false);
                return true;
            }
            return blockExecutor.breakBlock(
                trigger.player(),
                block,
                trigger.action(),
                trigger.durabilityMode()
            );
        } finally {
            generatedBreaks.remove(playerId);
        }
    }

    private List<Animals> animalTargets(AnimalTrigger trigger) {
        BatchSearchSpace searchSpace = new BatchSearchSpace(
            trigger.origin(),
            trigger.setting(),
            trigger.world().getMinHeight(),
            trigger.world().getMaxHeight()
        );
        List<Animals> targets = new ArrayList<>();
        for (Chunk chunk : relevantChunks(
            trigger.player(), trigger.world(), searchSpace, trigger.origin()
        )) {
            for (Entity entity : chunk.getEntities()) {
                if (entity instanceof Animals animal
                    && animal.getType() == trigger.type()
                    && !animal.getUniqueId().equals(trigger.originalId())
                    && animal.isValid()
                    && !animal.isDead()
                    && searchSpace.contains(blockPosition(animal))) {
                    targets.add(animal);
                }
            }
        }
        return targets;
    }

    private boolean damageAnimal(Player player, Animals animal) {
        if (!animal.isValid() || animal.isDead()) {
            return false;
        }
        UUID playerId = player.getUniqueId();
        generatedAnimalDamage.add(playerId);
        try {
            animal.damage(GENERATED_ANIMAL_DAMAGE, player);
            return animal.isDead() || animal.getHealth() <= 0.0;
        } finally {
            generatedAnimalDamage.remove(playerId);
        }
    }

    private Player attackingPlayer(Entity damager) {
        if (damager instanceof Player player) {
            return player;
        }
        if (damager instanceof Projectile projectile) {
            ProjectileSource shooter = projectile.getShooter();
            if (shooter instanceof Player player) {
                return player;
            }
        }
        return null;
    }

    private BlockPosition blockPosition(Entity entity) {
        var location = entity.getLocation();
        return new BlockPosition(
            location.getBlockX(),
            location.getBlockY(),
            location.getBlockZ()
        );
    }

    private void finish(Player player, ActiveJob job, String target, long changed) {
        activeJobs.remove(player.getUniqueId(), job);
        plugin.getLogger().fine(
            () -> "Batch " + target + " completed with " + changed + " changes"
        );
    }

    private boolean isPlayerValid(Player player, World world) {
        return player.isOnline() && player.getWorld().equals(world);
    }

    private interface ActiveJob {
        void start();

        void cancel();
    }

    @FunctionalInterface
    private interface PositionSink {
        void accept(BlockPosition position) throws InterruptedException;
    }

    private final class SynchronousBlockJob implements ActiveJob {
        private final BlockTrigger trigger;
        private final BatchSearchSpace searchSpace;
        private boolean cancelled;
        private long changed;

        private SynchronousBlockJob(BlockTrigger trigger) {
            this.trigger = trigger;
            searchSpace = new BatchSearchSpace(
                trigger.origin(),
                trigger.setting(),
                trigger.minimumY(),
                trigger.maximumY()
            );
        }

        @Override
        public void start() {
            try {
                for (Chunk chunk : relevantChunks(
                    trigger.player(), trigger.world(), searchSpace, trigger.origin()
                )) {
                    if (cancelled || !isPlayerValid(trigger.player(), trigger.world())) {
                        return;
                    }
                    ChunkSnapshot snapshot = chunk.getChunkSnapshot(
                        false, false, false, false
                    );
                    scanSnapshot(snapshot, trigger, searchSpace, position -> {
                        if (!cancelled && applyBlock(trigger, position)) {
                            changed++;
                        }
                    });
                }
            } catch (InterruptedException impossible) {
                Thread.currentThread().interrupt();
            } finally {
                finish(trigger.player(), this, "block", changed);
            }
        }

        @Override
        public void cancel() {
            cancelled = true;
        }
    }

    private final class AsynchronousBlockJob implements ActiveJob {
        private final BlockTrigger trigger;
        private final BatchSearchSpace searchSpace;
        private final ArrayDeque<Chunk> chunks;
        private final ArrayBlockingQueue<BlockPosition> matches =
            new ArrayBlockingQueue<>(ASYNC_MATCH_QUEUE_CAPACITY);
        private final AtomicInteger scansInFlight = new AtomicInteger();
        private final ArrayList<Future<?>> futures = new ArrayList<>();
        private BukkitTask captureTask;
        private BukkitTask breakTask;
        private boolean captureComplete;
        private volatile boolean cancelled;
        private long changed;

        private AsynchronousBlockJob(BlockTrigger trigger) {
            this.trigger = trigger;
            searchSpace = new BatchSearchSpace(
                trigger.origin(),
                trigger.setting(),
                trigger.minimumY(),
                trigger.maximumY()
            );
            chunks = new ArrayDeque<>(
                relevantChunks(
                    trigger.player(), trigger.world(), searchSpace, trigger.origin()
                )
            );
        }

        @Override
        public void start() {
            captureTask = Bukkit.getScheduler().runTaskTimer(
                plugin, this::captureSnapshots, 1L, 1L
            );
            breakTask = Bukkit.getScheduler().runTaskTimer(
                plugin, this::breakMatches, 1L, 1L
            );
        }

        private void captureSnapshots() {
            if (!isPlayerValid(trigger.player(), trigger.world())) {
                cancel();
                return;
            }
            int budget = ASYNC_SNAPSHOTS_PER_TICK;
            while (budget-- > 0 && scansInFlight.get() < ASYNC_SNAPSHOTS_PER_TICK) {
                Chunk chunk = chunks.pollFirst();
                if (chunk == null) {
                    captureComplete = true;
                    captureTask.cancel();
                    return;
                }
                if (!trigger.player().isChunkSent(chunk)
                    || !trigger.world().isChunkLoaded(chunk.getX(), chunk.getZ())) {
                    continue;
                }
                ChunkSnapshot snapshot = chunk.getChunkSnapshot(
                    false, false, false, false
                );
                scansInFlight.incrementAndGet();
                futures.add(scanner.submit(() -> scan(snapshot)));
            }
        }

        private void scan(ChunkSnapshot snapshot) {
            try {
                scanSnapshot(snapshot, trigger, searchSpace, matches::put);
            } catch (InterruptedException interrupted) {
                Thread.currentThread().interrupt();
            } finally {
                scansInFlight.decrementAndGet();
            }
        }

        private void breakMatches() {
            if (!isPlayerValid(trigger.player(), trigger.world())) {
                cancel();
                return;
            }
            int budget = ASYNC_CHANGES_PER_TICK;
            while (budget-- > 0) {
                BlockPosition position = matches.poll();
                if (position == null) {
                    break;
                }
                if (applyBlock(trigger, position)) {
                    changed++;
                }
            }
            if (captureComplete && scansInFlight.get() == 0 && matches.isEmpty()) {
                cancelTasks();
                finish(trigger.player(), this, "block", changed);
            }
        }

        @Override
        public void cancel() {
            cancelled = true;
            cancelTasks();
            for (Future<?> future : futures) {
                future.cancel(true);
            }
            activeJobs.remove(trigger.player().getUniqueId(), this);
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

    private final class SynchronousAnimalJob implements ActiveJob {
        private final Player player;
        private final List<Animals> targets;
        private boolean cancelled;
        private long changed;

        private SynchronousAnimalJob(Player player, List<Animals> targets) {
            this.player = player;
            this.targets = targets;
        }

        @Override
        public void start() {
            try {
                for (Animals animal : targets) {
                    if (cancelled || !player.isOnline()) {
                        return;
                    }
                    if (damageAnimal(player, animal)) {
                        changed++;
                    }
                }
            } finally {
                finish(player, this, "animal", changed);
            }
        }

        @Override
        public void cancel() {
            cancelled = true;
        }
    }

    private final class AsynchronousAnimalJob implements ActiveJob {
        private final Player player;
        private final ArrayDeque<Animals> targets;
        private BukkitTask task;
        private boolean cancelled;
        private long changed;

        private AsynchronousAnimalJob(Player player, List<Animals> targets) {
            this.player = player;
            this.targets = new ArrayDeque<>(targets);
        }

        @Override
        public void start() {
            task = Bukkit.getScheduler().runTaskTimer(
                plugin, this::damageTargets, 1L, 1L
            );
        }

        private void damageTargets() {
            if (cancelled || !player.isOnline()) {
                cancel();
                return;
            }
            int budget = ASYNC_CHANGES_PER_TICK;
            while (budget-- > 0) {
                Animals animal = targets.pollFirst();
                if (animal == null) {
                    task.cancel();
                    finish(player, this, "animal", changed);
                    return;
                }
                if (damageAnimal(player, animal)) {
                    changed++;
                }
            }
        }

        @Override
        public void cancel() {
            cancelled = true;
            if (task != null && !task.isCancelled()) {
                task.cancel();
            }
            activeJobs.remove(player.getUniqueId(), this);
        }
    }

    private record BlockTrigger(
        Player player,
        World world,
        Material material,
        BlockPosition origin,
        BatchBreakingSetting setting,
        BatchBlockAction action,
        ToolDurabilityMode durabilityMode,
        int minimumY,
        int maximumY,
        boolean fluid
    ) {
    }

    private record AnimalTrigger(
        Player player,
        World world,
        EntityType type,
        UUID originalId,
        BlockPosition origin,
        BatchBreakingSetting setting,
        Animals original
    ) {
    }
}
