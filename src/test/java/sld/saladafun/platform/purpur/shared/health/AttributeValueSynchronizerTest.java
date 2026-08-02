package sld.saladafun.platform.purpur.shared.health;

import org.bukkit.attribute.AttributeInstance;
import org.bukkit.attribute.AttributeModifier;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.concurrent.atomic.AtomicReference;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class AttributeValueSynchronizerTest {

    @Test
    void preservesModifiersWhileSolvingTheRequiredBaseValue() {
        AttributeInstance attribute = mock(AttributeInstance.class);
        AttributeModifier additive = modifier(
            2.0, AttributeModifier.Operation.ADD_NUMBER
        );
        AttributeModifier scalar = modifier(
            0.5, AttributeModifier.Operation.ADD_SCALAR
        );
        AttributeModifier multiplier = modifier(
            0.25, AttributeModifier.Operation.MULTIPLY_SCALAR_1
        );
        AtomicReference<Double> base = new AtomicReference<>(10.0);
        when(attribute.getModifiers()).thenReturn(List.of(
            additive, scalar, multiplier
        ));
        doAnswer(invocation -> {
            base.set(invocation.getArgument(0));
            return null;
        }).when(attribute).setBaseValue(org.mockito.ArgumentMatchers.anyDouble());
        when(attribute.getValue()).thenAnswer(invocation ->
            (base.get() + 2.0) * 1.5 * 1.25
        );

        new AttributeValueSynchronizer().setEffectiveValue(attribute, 30.0);

        assertEquals(14.0, base.get(), 1.0E-9);
        assertEquals(30.0, attribute.getValue(), 1.0E-9);
    }

    @Test
    void rejectsModifierSetsThatCollapseTheRange() {
        AttributeInstance attribute = mock(AttributeInstance.class);
        AttributeModifier collapsing = modifier(
            -1.0, AttributeModifier.Operation.ADD_SCALAR
        );
        when(attribute.getModifiers()).thenReturn(List.of(collapsing));

        assertThrows(
            IllegalStateException.class,
            () -> new AttributeValueSynchronizer().setEffectiveValue(attribute, 20.0)
        );
    }

    private AttributeModifier modifier(
        double amount,
        AttributeModifier.Operation operation
    ) {
        AttributeModifier modifier = mock(AttributeModifier.class);
        when(modifier.getAmount()).thenReturn(amount);
        when(modifier.getOperation()).thenReturn(operation);
        return modifier;
    }
}
