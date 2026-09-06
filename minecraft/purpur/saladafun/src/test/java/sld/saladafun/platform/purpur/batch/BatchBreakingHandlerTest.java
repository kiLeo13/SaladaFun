package sld.saladafun.platform.purpur.batch;

import org.bukkit.Bukkit;
import org.bukkit.Chunk;
import org.bukkit.ChunkSnapshot;
import org.bukkit.Location;
import org.bukkit.Material;
import org.bukkit.World;
import org.bukkit.block.Block;
import org.bukkit.entity.Animals;
import org.bukkit.entity.Entity;
import org.bukkit.entity.EntityType;
import org.bukkit.entity.Player;
import org.bukkit.event.block.BlockBreakEvent;
import org.bukkit.event.block.BlockDropItemEvent;
import org.bukkit.event.entity.EntityDamageByEntityEvent;
import org.bukkit.event.player.PlayerBucketFillEvent;
import org.bukkit.plugin.PluginManager;
import org.bukkit.plugin.java.JavaPlugin;
import org.bukkit.configuration.file.YamlConfiguration;
import org.bukkit.scheduler.BukkitScheduler;
import org.bukkit.scheduler.BukkitTask;
import org.junit.jupiter.api.Test;
import org.mockito.MockedStatic;
import sld.saladafun.platform.purpur.config.PluginSettings;

import java.util.Set;
import java.util.UUID;
import java.util.logging.Logger;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyBoolean;
import static org.mockito.ArgumentMatchers.anyDouble;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.atLeastOnce;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.mockStatic;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class BatchBreakingHandlerTest {

    @Test
    void synchronousRadiusOneImmediatelyBreaksAnAdjacentMatchingBlock() {
        Fixture fixture = new Fixture(false, "SYNC");
        Block neighbor = mock(Block.class);
        when(neighbor.getType()).thenReturn(Material.STONE);
        when(fixture.world.getBlockAt(9, 64, 8)).thenReturn(neighbor);
        when(fixture.snapshot.getBlockType(9, 64, 8)).thenReturn(Material.STONE);
        when(fixture.player.breakBlock(neighbor)).thenReturn(true);

        BlockBreakEvent event = mock(BlockBreakEvent.class);
        when(event.getPlayer()).thenReturn(fixture.player);
        when(event.getBlock()).thenReturn(fixture.origin);

        try (MockedStatic<Bukkit> bukkit = mockStatic(Bukkit.class);
             var handler = fixture.handler()) {
            handler.captureBlockTrigger(event);
            handler.startAfterBlockOutcome(event);
            bukkit.verify(Bukkit::getScheduler, never());
        }

        verify(fixture.player).breakBlock(neighbor);
        verify(fixture.player, never()).sendMessage(any(String.class));
    }

    @Test
    void bucketFillRemovesAdjacentWaterAfterGeneratedBreakChecks() {
        assertBucketFillRemovesAdjacentFluid(Material.WATER);
    }

    @Test
    void bucketFillRemovesAdjacentLavaAfterGeneratedBreakChecks() {
        assertBucketFillRemovesAdjacentFluid(Material.LAVA);
    }

    private void assertBucketFillRemovesAdjacentFluid(Material fluidMaterial) {
        Fixture fixture = new Fixture(false, "SYNC");
        Block fluid = mock(Block.class);
        when(fluid.getType()).thenReturn(fluidMaterial);
        when(fixture.origin.getType()).thenReturn(fluidMaterial);
        when(fixture.world.getBlockAt(9, 64, 8)).thenReturn(fluid);
        when(fixture.snapshot.getBlockType(9, 64, 8)).thenReturn(fluidMaterial);

        PlayerBucketFillEvent event = mock(PlayerBucketFillEvent.class);
        when(event.getPlayer()).thenReturn(fixture.player);
        when(event.getBlock()).thenReturn(fixture.origin);
        PluginManager pluginManager = mock(PluginManager.class);

        try (MockedStatic<Bukkit> bukkit = mockStatic(Bukkit.class);
             var handler = fixture.handler()) {
            bukkit.when(Bukkit::getPluginManager).thenReturn(pluginManager);
            handler.captureBucketTrigger(event);
            handler.startAfterBucketOutcome(event);
        }

        verify(pluginManager).callEvent(any(BlockBreakEvent.class));
        verify(fluid).setType(Material.AIR, false);
        verify(fixture.player, never()).sendMessage(any(String.class));
    }

    @Test
    void bucketFillHonorsCancellationOfGeneratedWaterBreaks() {
        Fixture fixture = new Fixture(false, "SYNC");
        Block water = mock(Block.class);
        when(water.getType()).thenReturn(Material.WATER);
        when(fixture.origin.getType()).thenReturn(Material.WATER);
        when(fixture.world.getBlockAt(9, 64, 8)).thenReturn(water);
        when(fixture.snapshot.getBlockType(9, 64, 8)).thenReturn(Material.WATER);

        PlayerBucketFillEvent event = mock(PlayerBucketFillEvent.class);
        when(event.getPlayer()).thenReturn(fixture.player);
        when(event.getBlock()).thenReturn(fixture.origin);
        PluginManager pluginManager = mock(PluginManager.class);
        doAnswer(invocation -> {
            ((BlockBreakEvent) invocation.getArgument(0)).setCancelled(true);
            return null;
        }).when(pluginManager).callEvent(any(BlockBreakEvent.class));

        try (MockedStatic<Bukkit> bukkit = mockStatic(Bukkit.class);
             var handler = fixture.handler()) {
            bukkit.when(Bukkit::getPluginManager).thenReturn(pluginManager);
            handler.captureBucketTrigger(event);
            handler.startAfterBucketOutcome(event);
        }

        verify(water, never()).setType(any(Material.class), anyBoolean());
    }

    @Test
    void animalBatchTargetsOnlySameSpeciesInsideTheConfiguredRange() {
        Fixture fixture = new Fixture(true, "SYNC");
        Animals original = animal(EntityType.PIG, fixture.world, 8, 64, 8);
        Animals nearbyPig = animal(EntityType.PIG, fixture.world, 9, 64, 8);
        Animals distantPig = animal(EntityType.PIG, fixture.world, 12, 64, 8);
        Animals nearbyCow = animal(EntityType.COW, fixture.world, 9, 64, 8);
        when(original.getHealth()).thenReturn(5.0);
        when(nearbyPig.isDead()).thenReturn(false, false, true);
        when(fixture.chunk.getEntities()).thenReturn(
            new Entity[]{original, nearbyPig, distantPig, nearbyCow}
        );

        EntityDamageByEntityEvent event = mock(EntityDamageByEntityEvent.class);
        when(event.getEntity()).thenReturn(original);
        when(event.getDamager()).thenReturn(fixture.player);
        when(event.getFinalDamage()).thenReturn(5.0);

        try (var handler = fixture.handler()) {
            handler.captureAnimalTrigger(event);
            handler.startAfterAnimalOutcome(event);
        }

        verify(nearbyPig).damage(1_000_000.0, fixture.player);
        verify(distantPig, never()).damage(anyDouble(), any(Entity.class));
        verify(nearbyCow, never()).damage(anyDouble(), any(Entity.class));
        verify(fixture.player, never()).sendMessage(any(String.class));
    }

    @Test
    void asynchronousModeSchedulesIncrementalMainThreadWork() {
        Fixture fixture = new Fixture(false, "ASYNC");
        BlockBreakEvent event = mock(BlockBreakEvent.class);
        when(event.getPlayer()).thenReturn(fixture.player);
        when(event.getBlock()).thenReturn(fixture.origin);
        BukkitScheduler scheduler = mock(BukkitScheduler.class);
        BukkitTask captureTask = mock(BukkitTask.class);
        BukkitTask breakTask = mock(BukkitTask.class);
        when(scheduler.runTaskTimer(
            any(JavaPlugin.class), any(Runnable.class), anyLong(), anyLong()
        )).thenReturn(captureTask, breakTask);

        try (MockedStatic<Bukkit> bukkit = mockStatic(Bukkit.class);
             var handler = fixture.handler()) {
            bukkit.when(Bukkit::getScheduler).thenReturn(scheduler);
            handler.captureBlockTrigger(event);
            handler.startAfterBlockOutcome(event);
            verify(scheduler, times(2)).runTaskTimer(
                eq(fixture.plugin), any(Runnable.class), eq(1L), eq(1L)
            );
        }

        verify(fixture.player, never()).sendMessage(any(String.class));
    }

    @Test
    void noDropsIsEnforcedForBothBreakAndFinalDropEvents() {
        Fixture fixture = new Fixture(false, "SYNC", "NO_DROPS");
        Block neighbor = mock(Block.class);
        when(neighbor.getType()).thenReturn(Material.STONE);
        when(fixture.world.getBlockAt(9, 64, 8)).thenReturn(neighbor);
        when(fixture.snapshot.getBlockType(9, 64, 8)).thenReturn(Material.STONE);
        BlockBreakEvent generatedBreak = mock(BlockBreakEvent.class);
        when(generatedBreak.getPlayer()).thenReturn(fixture.player);
        BlockDropItemEvent generatedDrops = mock(BlockDropItemEvent.class);
        when(generatedDrops.getPlayer()).thenReturn(fixture.player);

        BlockBreakEvent originalBreak = mock(BlockBreakEvent.class);
        when(originalBreak.getPlayer()).thenReturn(fixture.player);
        when(originalBreak.getBlock()).thenReturn(fixture.origin);

        try (var handler = fixture.handler()) {
            when(fixture.player.breakBlock(neighbor)).thenAnswer(invocation -> {
                handler.captureBlockTrigger(generatedBreak);
                handler.enforceGeneratedBreakPolicy(generatedBreak);
                handler.suppressGeneratedDrops(generatedDrops);
                return true;
            });
            handler.captureBlockTrigger(originalBreak);
            handler.startAfterBlockOutcome(originalBreak);
        }

        verify(generatedBreak, atLeastOnce()).setDropItems(false);
        verify(generatedBreak, atLeastOnce()).setExpToDrop(0);
        verify(generatedDrops).setCancelled(true);
    }

    private Animals animal(
        EntityType type,
        World world,
        int x,
        int y,
        int z
    ) {
        Animals animal = mock(Animals.class);
        Location location = mock(Location.class);
        when(location.getBlockX()).thenReturn(x);
        when(location.getBlockY()).thenReturn(y);
        when(location.getBlockZ()).thenReturn(z);
        when(animal.getLocation()).thenReturn(location);
        when(animal.getWorld()).thenReturn(world);
        when(animal.getType()).thenReturn(type);
        when(animal.getUniqueId()).thenReturn(UUID.randomUUID());
        when(animal.isValid()).thenReturn(true);
        when(animal.isDead()).thenReturn(false);
        return animal;
    }

    private static final class Fixture {
        private final JavaPlugin plugin = mock(JavaPlugin.class);
        private final Player player = mock(Player.class);
        private final World world = mock(World.class);
        private final Chunk chunk = mock(Chunk.class);
        private final ChunkSnapshot snapshot = mock(ChunkSnapshot.class);
        private final Block origin = mock(Block.class);
        private final PluginSettings settings;

        private Fixture(boolean includeAnimals, String executionMode) {
            this(includeAnimals, executionMode, "PLAYER_TOOL");
        }

        private Fixture(
            boolean includeAnimals,
            String executionMode,
            String blockAction
        ) {
            YamlConfiguration configuration = new YamlConfiguration();
            configuration.set("batch-breaking.setting", "1");
            configuration.set("batch-breaking.sync-batching", executionMode);
            configuration.set("batch-breaking.additional-block-action", blockAction);
            configuration.set("batch-breaking.tool-durability", "PER_BLOCK");
            configuration.set("batch-breaking.include-animals", includeAnimals);
            settings = new PluginSettings(configuration);

            when(plugin.getLogger()).thenReturn(Logger.getAnonymousLogger());
            when(player.getUniqueId()).thenReturn(UUID.randomUUID());
            when(player.hasPermission("saladafun.batchbreaking.use")).thenReturn(true);
            when(player.isOnline()).thenReturn(true);
            when(player.getWorld()).thenReturn(world);
            when(player.getSentChunks()).thenReturn(Set.of(chunk));
            when(player.isChunkSent(chunk)).thenReturn(true);

            when(world.getMinHeight()).thenReturn(-64);
            when(world.getMaxHeight()).thenReturn(320);
            when(world.isChunkLoaded(0, 0)).thenReturn(true);
            when(world.getChunkAt(0, 0)).thenReturn(chunk);

            when(chunk.getWorld()).thenReturn(world);
            when(chunk.getX()).thenReturn(0);
            when(chunk.getZ()).thenReturn(0);
            when(chunk.getChunkSnapshot(false, false, false, false))
                .thenReturn(snapshot);

            when(snapshot.getX()).thenReturn(0);
            when(snapshot.getZ()).thenReturn(0);

            when(origin.getWorld()).thenReturn(world);
            when(origin.getType()).thenReturn(Material.STONE);
            when(origin.getX()).thenReturn(8);
            when(origin.getY()).thenReturn(64);
            when(origin.getZ()).thenReturn(8);
        }

        private BatchBreakingHandler handler() {
            return new BatchBreakingHandler(plugin, settings);
        }
    }
}
