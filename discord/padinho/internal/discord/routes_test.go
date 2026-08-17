package discord

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
)

func TestRoutesDispatchComponentsAndModals(t *testing.T) {
	routes := NewRoutes()
	var componentRequest, modalRequest *InteractionRequest
	routes.Component("birthday.page", func(_ context.Context, request *InteractionRequest) error {
		componentRequest = request
		return nil
	})
	routes.Modal("birthday.add", func(_ context.Context, request *InteractionRequest) error {
		modalRequest = request
		return nil
	})
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	responder := &testResponder{}
	component := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionMessageComponent, GuildID: "guild", ChannelID: "channel",
		Member: &discordgo.Member{User: &discordgo.User{ID: "123"}, Roles: []string{"role"}, Permissions: discordgo.PermissionManageGuild},
		Data:   discordgo.MessageComponentInteractionData{CustomID: "birthday.page:next:1"},
	}}
	handled, err := routes.dispatch(context.Background(), component, responder)
	if err != nil || !handled || componentRequest == nil || !reflect.DeepEqual(componentRequest.Parameters, []string{"next", "1"}) || componentRequest.Actor.UserID != "123" || componentRequest.Actor.Permissions != discordgo.PermissionManageGuild || componentRequest.GuildID != "guild" || componentRequest.ChannelID != "channel" {
		t.Fatalf("component request = %#v, handled = %v, error = %v", componentRequest, handled, err)
	}
	modal := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionModalSubmit,
		User: &discordgo.User{ID: "dm"},
		Data: discordgo.ModalSubmitInteractionData{CustomID: "birthday.add"},
	}}
	handled, err = routes.dispatch(context.Background(), modal, responder)
	if err != nil || !handled || modalRequest == nil || modalRequest.Actor.UserID != "dm" {
		t.Fatalf("modal request = %#v, handled = %v, error = %v", modalRequest, handled, err)
	}
}

func TestRoutesDispatchCommandsAndIgnoreOtherEvents(t *testing.T) {
	routes := NewRoutes()
	called := false
	routes.Commands().Slash("ping", "Ping", func(context.Context, *command.CommandRequest) error {
		called = true
		return nil
	})
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	interaction := interactionWithOptions(nil)
	interaction.Data = discordgo.ApplicationCommandInteractionData{Name: "ping"}
	handled, err := routes.dispatch(context.Background(), interaction, &testResponder{})
	if err != nil || !handled || !called {
		t.Fatalf("command handled = %v, called = %v, error = %v", handled, called, err)
	}
	handled, err = routes.dispatch(context.Background(), &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Type: discordgo.InteractionPing}}, &testResponder{})
	if err != nil || handled {
		t.Fatalf("ping handled = %v, error = %v", handled, err)
	}
}

func TestRoutesLifecycleAndUnknownRouteErrors(t *testing.T) {
	routes := NewRoutes()
	handled, err := routes.dispatch(context.Background(), interactionWithOptions(nil), &testResponder{})
	if !handled || !errors.Is(err, ErrRoutesNotFrozen) {
		t.Fatalf("unfrozen dispatch = %v, %v", handled, err)
	}
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	if err := routes.Freeze(); err != nil {
		t.Fatalf("second Freeze() error = %v", err)
	}
	unknown := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionMessageComponent,
		Data: discordgo.MessageComponentInteractionData{CustomID: "missing:value"},
	}}
	if _, err := routes.dispatch(context.Background(), unknown, &testResponder{}); !errors.Is(err, ErrUnknownRoute) {
		t.Fatalf("unknown route error = %v", err)
	}
}

func TestRoutesPropagateCommandFreezeAndMappingErrors(t *testing.T) {
	invalid := NewRoutes()
	invalid.Commands().Slash("UPPER", "Invalid", func(context.Context, *command.CommandRequest) error { return nil })
	if err := invalid.Freeze(); err == nil {
		t.Fatal("invalid command registry froze")
	}
	routes := NewRoutes()
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	malformed := interactionWithOptions([]*discordgo.ApplicationCommandInteractionDataOption{{
		Name: "group", Type: discordgo.ApplicationCommandOptionSubCommandGroup,
	}})
	handled, err := routes.dispatch(context.Background(), malformed, &testResponder{})
	if !handled || err == nil {
		t.Fatalf("malformed dispatch = %v, %v", handled, err)
	}
}

func TestRoutesRejectInvalidRegistration(t *testing.T) {
	handler := func(context.Context, *InteractionRequest) error { return nil }
	tests := map[string]func(*Routes){
		"empty":     func(routes *Routes) { routes.Component("", handler) },
		"separator": func(routes *Routes) { routes.Component("bad:route", handler) },
		"nil":       func(routes *Routes) { routes.Component("route", nil) },
		"duplicate": func(routes *Routes) { routes.Component("route", handler); routes.Component("route", handler) },
		"frozen":    func(routes *Routes) { _ = routes.Freeze(); routes.Modal("route", handler) },
	}
	for name, operation := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("registration did not panic")
				}
			}()
			operation(NewRoutes())
		})
	}
}
