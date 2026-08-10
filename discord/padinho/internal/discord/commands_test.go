package discord

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
)

func TestCompileDefinitions(t *testing.T) {
	t.Parallel()
	definitions := []command.Definition{
		{Name: "ping", Description: "Check", Options: allOptions()},
		{Name: "admin", Description: "Admin", Subcommands: []command.SubcommandDefinition{{Name: "ban", Description: "Ban"}}, Groups: []command.SubcommandGroupDefinition{{Name: "members", Description: "Members", Subcommands: []command.SubcommandDefinition{{Name: "add", Description: "Add", Options: allOptions()}}}}},
	}
	compiled, err := CompileDefinitions(definitions)
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled) != 2 || compiled[0].Type != discordgo.ChatApplicationCommand || len(compiled[0].Options) != 5 || compiled[1].Options[0].Type != discordgo.ApplicationCommandOptionSubCommand || compiled[1].Options[1].Type != discordgo.ApplicationCommandOptionSubCommandGroup {
		t.Fatalf("CompileDefinitions() = %#v", compiled)
	}
	if compiled[0].Options[0].Type != discordgo.ApplicationCommandOptionString || !compiled[0].Options[0].Required || !compiled[0].Options[0].Autocomplete {
		t.Fatalf("compiled string option = %#v", compiled[0].Options[0])
	}
}

func TestCompileDefinitionsRejectsUnknownOptionAtEveryLevel(t *testing.T) {
	t.Parallel()
	tests := []command.Definition{
		{Name: "root", Description: "Root", Options: invalidOptions()},
		{Name: "root", Description: "Root", Subcommands: []command.SubcommandDefinition{{Name: "sub", Description: "Sub", Options: invalidOptions()}}},
		{Name: "root", Description: "Root", Groups: []command.SubcommandGroupDefinition{{Name: "group", Description: "Group", Subcommands: []command.SubcommandDefinition{{Name: "sub", Description: "Sub", Options: invalidOptions()}}}}},
	}
	for _, definition := range tests {
		if _, err := CompileDefinitions([]command.Definition{definition}); err == nil || !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("CompileDefinitions(%#v) error = %v", definition, err)
		}
	}
}

func TestMapRequest(t *testing.T) {
	t.Parallel()
	interaction := interactionWithOptions([]*discordgo.ApplicationCommandInteractionDataOption{{
		Name: "members", Type: discordgo.ApplicationCommandOptionSubCommandGroup,
		Options: []*discordgo.ApplicationCommandInteractionDataOption{{
			Name: "add", Type: discordgo.ApplicationCommandOptionSubCommand,
			Options: []*discordgo.ApplicationCommandInteractionDataOption{
				{Name: "name", Type: discordgo.ApplicationCommandOptionString, Value: "Leo"},
				{Name: "count", Type: discordgo.ApplicationCommandOptionInteger, Value: float64(2)},
				{Name: "quiet", Type: discordgo.ApplicationCommandOptionBoolean, Value: true},
				{Name: "user", Type: discordgo.ApplicationCommandOptionUser, Value: "456"},
				{Name: "channel", Type: discordgo.ApplicationCommandOptionChannel, Value: "789"},
			},
		}},
	}})
	request, responder, err := mapRequest(&discordgo.Session{}, interaction)
	if err != nil {
		t.Fatal(err)
	}
	if request.Path != (command.CommandPath{Command: "groups", Group: "members", Subcommand: "add"}) || request.Actor.UserID != "123" || !reflect.DeepEqual(request.Actor.RoleIDs, []command.Snowflake{"role"}) || request.GuildID != "guild" || request.ChannelID != "channel" || request.Responder != responder || request.RequestID != "request" {
		t.Fatalf("mapRequest() = %#v", request)
	}
	if value, _ := request.Options.Integer("count"); value != 2 {
		t.Fatalf("count = %d", value)
	}
	if value, _ := request.Options.Snowflake("user"); value != "456" {
		t.Fatalf("user = %q", value)
	}
}

func TestMapRequestDirectAndMalformedOptions(t *testing.T) {
	t.Parallel()
	direct := interactionWithOptions([]*discordgo.ApplicationCommandInteractionDataOption{{Name: "ban", Type: discordgo.ApplicationCommandOptionSubCommand}})
	request, _, err := mapRequest(&discordgo.Session{}, direct)
	if err != nil || request.Path.Subcommand != "ban" {
		t.Fatalf("direct map = %#v, %v", request, err)
	}

	dm := interactionWithOptions(nil)
	dm.Member = nil
	dm.User = &discordgo.User{ID: "dm-user"}
	request, _, err = mapRequest(&discordgo.Session{}, dm)
	if err != nil || request.Actor.UserID != "dm-user" {
		t.Fatalf("DM map = %#v, %v", request, err)
	}

	malformed := interactionWithOptions([]*discordgo.ApplicationCommandInteractionDataOption{{Name: "group", Type: discordgo.ApplicationCommandOptionSubCommandGroup}})
	if _, _, err := mapRequest(&discordgo.Session{}, malformed); err == nil {
		t.Fatal("malformed group accepted")
	}
	unsupported := interactionWithOptions([]*discordgo.ApplicationCommandInteractionDataOption{{Name: "number", Type: discordgo.ApplicationCommandOptionNumber, Value: 1.5}})
	if _, _, err := mapRequest(&discordgo.Session{}, unsupported); err == nil {
		t.Fatal("unsupported option accepted")
	}
}

func allOptions() []command.OptionDefinition {
	return []command.OptionDefinition{
		{Type: command.OptionTypeString, Name: "string", Description: "String", Required: true, Autocomplete: true},
		{Type: command.OptionTypeInteger, Name: "integer", Description: "Integer"},
		{Type: command.OptionTypeBoolean, Name: "boolean", Description: "Boolean"},
		{Type: command.OptionTypeUser, Name: "user", Description: "User"},
		{Type: command.OptionTypeChannel, Name: "channel", Description: "Channel"},
	}
}

func invalidOptions() []command.OptionDefinition {
	return []command.OptionDefinition{{Type: 255, Name: "bad", Description: "Bad"}}
}

func interactionWithOptions(options []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID: "request", Type: discordgo.InteractionApplicationCommand,
		GuildID: "guild", ChannelID: "channel",
		Member: &discordgo.Member{User: &discordgo.User{ID: "123"}, Roles: []string{"role"}},
		Data:   discordgo.ApplicationCommandInteractionData{Name: "groups", Options: options},
	}}
}
