package move

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

func TestRegister(t *testing.T) {
	routes := discord.NewRoutes()
	Register(routes, &fakeService{})
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	definitions, err := routes.Commands().Definitions()
	if err != nil || len(definitions) != 1 {
		t.Fatalf("Definitions() = %#v, %v", definitions, err)
	}
	definition := definitions[0]
	if definition.Name != commandName || definition.Description != ptbr.MoveAllCommandDescription || len(definition.Options) != 2 {
		t.Fatalf("definition = %#v", definition)
	}
	if option := definition.Options[0]; option.Name != "destination" || option.Type != command.OptionTypeChannel || !option.Required {
		t.Fatalf("destination option = %#v", option)
	}
	if option := definition.Options[1]; option.Name != "origin" || option.Type != command.OptionTypeChannel || option.Required {
		t.Fatalf("origin option = %#v", option)
	}
}

func TestMoveAllUsesCallersVoiceChannelAndMovesEveryMember(t *testing.T) {
	service := &fakeService{callerChannel: "origin", voiceChannels: map[command.Snowflake]bool{"origin": true, "destination": true}, members: []command.Snowflake{"one", "two", "three"}}
	responder := &fakeResponder{}
	err := (Handler{service: service}).MoveAll(context.Background(), request(responder, map[string]any{"destination": command.Snowflake("destination")}))
	if err != nil {
		t.Fatal(err)
	}
	if service.currentCalls != 1 || !reflect.DeepEqual(service.moves, service.members) {
		t.Fatalf("current calls = %d, moves = %v", service.currentCalls, service.moves)
	}
	if got := responseText(responder.response); got != "Movendo 3 membro(s)." {
		t.Fatalf("response = %q", got)
	}
}

func TestMoveAllUsesExplicitOrigin(t *testing.T) {
	service := &fakeService{voiceChannels: map[command.Snowflake]bool{"origin": true, "destination": true}, members: []command.Snowflake{"one"}}
	err := (Handler{service: service}).MoveAll(context.Background(), request(&fakeResponder{}, map[string]any{"destination": command.Snowflake("destination"), "origin": command.Snowflake("origin")}))
	if err != nil {
		t.Fatal(err)
	}
	if service.currentCalls != 0 || !reflect.DeepEqual(service.moves, []command.Snowflake{"one"}) {
		t.Fatalf("current calls = %d, moves = %v", service.currentCalls, service.moves)
	}
}

func TestMoveAllRejectsInvalidRequests(t *testing.T) {
	tests := map[string]struct {
		service *fakeService
		options map[string]any
		want    string
	}{
		"caller not connected": {service: &fakeService{voiceChannels: map[command.Snowflake]bool{"destination": true}}, options: map[string]any{"destination": command.Snowflake("destination")}, want: ptbr.MoveAllOriginRequired},
		"invalid origin":       {service: &fakeService{voiceChannels: map[command.Snowflake]bool{"destination": true}}, options: map[string]any{"destination": command.Snowflake("destination"), "origin": command.Snowflake("text")}, want: ptbr.MoveAllInvalidOrigin},
		"invalid destination":  {service: &fakeService{voiceChannels: map[command.Snowflake]bool{"origin": true}}, options: map[string]any{"destination": command.Snowflake("text"), "origin": command.Snowflake("origin")}, want: ptbr.MoveAllInvalidDestination},
		"same channel":         {service: &fakeService{}, options: map[string]any{"destination": command.Snowflake("voice"), "origin": command.Snowflake("voice")}, want: ptbr.MoveAllSameChannel},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := (Handler{service: test.service}).MoveAll(context.Background(), request(&fakeResponder{}, test.options))
			rejection, ok := command.AsRejection(err)
			if !ok || rejection.Message != test.want {
				t.Fatalf("MoveAll() error = %v", err)
			}
		})
	}
}

func TestMoveAllAttemptsEveryMoveWhenOneFails(t *testing.T) {
	service := &fakeService{voiceChannels: map[command.Snowflake]bool{"origin": true, "destination": true}, members: []command.Snowflake{"one", "two", "three"}, moveErrors: map[command.Snowflake]error{"two": errors.New("forbidden")}}
	err := (Handler{service: service}).MoveAll(context.Background(), request(&fakeResponder{}, map[string]any{"destination": command.Snowflake("destination"), "origin": command.Snowflake("origin")}))
	if err == nil || !strings.Contains(err.Error(), "move member two") || !reflect.DeepEqual(service.moves, service.members) {
		t.Fatalf("MoveAll() error = %v, moves = %v", err, service.moves)
	}
}

func TestMoveAllReturnsOperationalErrors(t *testing.T) {
	serviceError := errors.New("service unavailable")
	tests := map[string]struct {
		service   *fakeService
		responder *fakeResponder
		options   map[string]any
	}{
		"missing destination":        {service: &fakeService{}, responder: &fakeResponder{}, options: map[string]any{}},
		"invalid origin option type": {service: &fakeService{}, responder: &fakeResponder{}, options: map[string]any{"destination": command.Snowflake("destination"), "origin": "origin"}},
		"caller voice lookup":        {service: &fakeService{currentErr: serviceError}, responder: &fakeResponder{}, options: map[string]any{"destination": command.Snowflake("destination")}},
		"channel validation":         {service: &fakeService{voiceErr: serviceError}, responder: &fakeResponder{}, options: map[string]any{"destination": command.Snowflake("destination"), "origin": command.Snowflake("origin")}},
		"member lookup":              {service: &fakeService{voiceChannels: map[command.Snowflake]bool{"origin": true, "destination": true}, membersErr: serviceError}, responder: &fakeResponder{}, options: map[string]any{"destination": command.Snowflake("destination"), "origin": command.Snowflake("origin")}},
		"interaction response":       {service: &fakeService{voiceChannels: map[command.Snowflake]bool{"origin": true, "destination": true}}, responder: &fakeResponder{err: serviceError}, options: map[string]any{"destination": command.Snowflake("destination"), "origin": command.Snowflake("origin")}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := (Handler{service: test.service}).MoveAll(context.Background(), request(test.responder, test.options))
			if !errors.Is(err, serviceError) && !errors.Is(err, command.ErrOptionMissing) && !errors.Is(err, command.ErrOptionType) {
				t.Fatalf("MoveAll() error = %v", err)
			}
		})
	}
}

func request(responder *fakeResponder, options map[string]any) *command.CommandRequest {
	return &command.CommandRequest{Actor: command.Actor{UserID: "caller"}, GuildID: "guild", Options: command.NewOptionValues(options), Responder: responder}
}

func responseText(response *discordgo.InteractionResponse) string {
	if response == nil || response.Data == nil || len(response.Data.Components) != 1 {
		return ""
	}
	text, ok := response.Data.Components[0].(discordgo.TextDisplay)
	if !ok {
		return ""
	}
	return text.Content
}

type fakeService struct {
	callerChannel command.Snowflake
	currentCalls  int
	currentErr    error
	voiceChannels map[command.Snowflake]bool
	voiceErr      error
	members       []command.Snowflake
	membersErr    error
	moves         []command.Snowflake
	moveErrors    map[command.Snowflake]error
}

func (s *fakeService) CurrentVoiceChannel(command.Snowflake, command.Snowflake) (command.Snowflake, bool, error) {
	s.currentCalls++
	return s.callerChannel, s.callerChannel != "", s.currentErr
}
func (s *fakeService) IsVoiceChannel(_ command.Snowflake, channelID command.Snowflake) (bool, error) {
	return s.voiceChannels[channelID], s.voiceErr
}
func (s *fakeService) MembersInVoiceChannel(command.Snowflake, command.Snowflake) ([]command.Snowflake, error) {
	return append([]command.Snowflake(nil), s.members...), s.membersErr
}
func (s *fakeService) MoveMember(_ command.Snowflake, userID, _ command.Snowflake) error {
	s.moves = append(s.moves, userID)
	return s.moveErrors[userID]
}

type fakeResponder struct {
	response *discordgo.InteractionResponse
	err      error
}

func (r *fakeResponder) Respond(response *discordgo.InteractionResponse) error {
	r.response = response
	return r.err
}
