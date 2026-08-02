package sld.saladafun.platform.purpur.shared;

import org.bukkit.command.Command;
import org.bukkit.command.CommandSender;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;

class SharedCommandTest {

    @Test
    void dispatchesToTheSelectedModuleAndCompletesModuleNames() {
        RecordingModule health = new RecordingModule("health");
        RecordingModule food = new RecordingModule("food");
        SharedCommand command = new SharedCommand(List.of(health, food));
        CommandSender sender = mock(CommandSender.class);

        command.onCommand(
            sender,
            mock(Command.class),
            "shared",
            new String[]{"health", "status"}
        );

        assertEquals(List.of("status"), health.arguments);
        assertEquals(
            List.of("food"),
            command.onTabComplete(
                sender, mock(Command.class), "shared", new String[]{"f"}
            )
        );
    }

    @Test
    void rejectsUnknownModulesWithoutCallingADelegate() {
        SharedCommand command = new SharedCommand(List.of(new RecordingModule("health")));
        CommandSender sender = mock(CommandSender.class);

        command.onCommand(
            sender,
            mock(Command.class),
            "shared",
            new String[]{"inventory", "enable"}
        );

        verify(sender).sendMessage("Unknown shared module: inventory");
    }

    private static final class RecordingModule implements SharedModuleCommand {
        private final String name;
        private List<String> arguments = List.of();

        private RecordingModule(String name) {
            this.name = name;
        }

        @Override
        public String moduleName() {
            return name;
        }

        @Override
        public boolean execute(CommandSender sender, String[] arguments) {
            this.arguments = List.of(arguments);
            return true;
        }

        @Override
        public List<String> complete(CommandSender sender, String[] arguments) {
            return List.of();
        }
    }
}
