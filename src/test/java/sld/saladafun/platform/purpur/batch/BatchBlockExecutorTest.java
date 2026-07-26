package sld.saladafun.platform.purpur.batch;

import org.bukkit.Material;
import org.bukkit.block.Block;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import org.bukkit.inventory.PlayerInventory;
import org.bukkit.inventory.meta.Damageable;
import org.junit.jupiter.api.Test;
import sld.saladafun.batchbreaking.BatchBlockAction;
import sld.saladafun.batchbreaking.ToolDurabilityMode;

import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;

class BatchBlockExecutorTest {
    private final BatchBlockExecutor executor = new BatchBlockExecutor();

    @Test
    void naturalDropsBypassPlayerAndToolDurability() {
        Player player = mock(Player.class);
        Block block = mock(Block.class);
        when(block.breakNaturally(true, true)).thenReturn(true);

        boolean broken = executor.breakBlock(
            player,
            block,
            BatchBlockAction.NATURAL_DROPS,
            ToolDurabilityMode.SINGLE_USE
        );

        assertTrue(broken);
        verify(block).breakNaturally(true, true);
        verifyNoInteractions(player);
    }

    @Test
    void perBlockLeavesPlayerToolDamageUntouched() {
        Player player = mock(Player.class);
        Block block = mock(Block.class);
        when(player.breakBlock(block)).thenReturn(true);

        boolean broken = executor.breakBlock(
            player,
            block,
            BatchBlockAction.PLAYER_TOOL,
            ToolDurabilityMode.PER_BLOCK
        );

        assertTrue(broken);
        verify(player).breakBlock(block);
        verify(player, never()).getInventory();
    }

    @Test
    void singleUseRestoresDamageAfterGeneratedPlayerBreak() {
        Player player = mock(Player.class);
        Block block = mock(Block.class);
        PlayerInventory inventory = mock(PlayerInventory.class);
        ItemStack before = mock(ItemStack.class);
        ItemStack snapshot = mock(ItemStack.class);
        ItemStack after = mock(ItemStack.class);
        Damageable beforeMetadata = mock(Damageable.class);
        Damageable afterMetadata = mock(Damageable.class);

        when(player.getInventory()).thenReturn(inventory);
        when(inventory.getItemInMainHand()).thenReturn(before, after);
        when(before.getItemMeta()).thenReturn(beforeMetadata);
        when(beforeMetadata.getDamage()).thenReturn(7);
        when(before.clone()).thenReturn(snapshot);
        when(after.getType()).thenReturn(Material.DIAMOND_PICKAXE);
        when(snapshot.getType()).thenReturn(Material.DIAMOND_PICKAXE);
        when(after.getItemMeta()).thenReturn(afterMetadata);
        when(player.breakBlock(block)).thenReturn(true);

        boolean broken = executor.breakBlock(
            player,
            block,
            BatchBlockAction.NO_DROPS,
            ToolDurabilityMode.SINGLE_USE
        );

        assertTrue(broken);
        verify(player).breakBlock(block);
        verify(afterMetadata).setDamage(7);
        verify(after).setItemMeta(afterMetadata);
        verify(inventory).setItemInMainHand(after);
    }

    @Test
    void singleUseRestoresToolIfGeneratedBreakWouldDestroyIt() {
        Player player = mock(Player.class);
        Block block = mock(Block.class);
        PlayerInventory inventory = mock(PlayerInventory.class);
        ItemStack before = mock(ItemStack.class);
        ItemStack snapshot = mock(ItemStack.class);
        ItemStack after = mock(ItemStack.class);
        Damageable beforeMetadata = mock(Damageable.class);

        when(player.getInventory()).thenReturn(inventory);
        when(inventory.getItemInMainHand()).thenReturn(before, after);
        when(before.getItemMeta()).thenReturn(beforeMetadata);
        when(beforeMetadata.getDamage()).thenReturn(1560);
        when(before.clone()).thenReturn(snapshot);
        when(after.getType()).thenReturn(Material.AIR);
        when(player.breakBlock(block)).thenReturn(true);

        boolean broken = executor.breakBlock(
            player,
            block,
            BatchBlockAction.PLAYER_TOOL,
            ToolDurabilityMode.SINGLE_USE
        );

        assertTrue(broken);
        verify(inventory).setItemInMainHand(snapshot);
        verify(after, never()).setItemMeta(beforeMetadata);
    }

    @Test
    void cancelledGeneratedBreakDoesNotRewriteTheHeldTool() {
        Player player = mock(Player.class);
        Block block = mock(Block.class);
        PlayerInventory inventory = mock(PlayerInventory.class);
        ItemStack before = mock(ItemStack.class);
        ItemStack snapshot = mock(ItemStack.class);
        Damageable beforeMetadata = mock(Damageable.class);

        when(player.getInventory()).thenReturn(inventory);
        when(inventory.getItemInMainHand()).thenReturn(before);
        when(before.getItemMeta()).thenReturn(beforeMetadata);
        when(beforeMetadata.getDamage()).thenReturn(20);
        when(before.clone()).thenReturn(snapshot);
        when(player.breakBlock(block)).thenReturn(false);

        boolean broken = executor.breakBlock(
            player,
            block,
            BatchBlockAction.PLAYER_TOOL,
            ToolDurabilityMode.SINGLE_USE
        );

        assertFalse(broken);
        verify(inventory, never()).setItemInMainHand(snapshot);
    }

    @Test
    void generatedPlayerBreakRestoresFoodSaturationAndExhaustion() {
        Player player = mock(Player.class);
        Block block = mock(Block.class);
        when(player.getFoodLevel()).thenReturn(17);
        when(player.getSaturation()).thenReturn(3.5F);
        when(player.getExhaustion()).thenReturn(2.25F);
        when(player.breakBlock(block)).thenReturn(true);

        boolean broken = executor.breakBlock(
            player,
            block,
            BatchBlockAction.NO_DROPS,
            ToolDurabilityMode.PER_BLOCK
        );

        assertTrue(broken);
        verify(player).setFoodLevel(17);
        verify(player).setSaturation(3.5F);
        verify(player).setExhaustion(2.25F);
    }
}
