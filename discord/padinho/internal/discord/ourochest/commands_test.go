package ourochest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

func TestCommandsToggleAutomaticAndStartManualHelpMidGame(t *testing.T) {
	listener, messenger := testListener(t)
	registry := messagecommand.NewRegistry()
	Register(registry, listener)
	if err := registry.Freeze(); err != nil {
		t.Fatal(err)
	}
	toggleResponder := &commandResponder{}
	handled, err := registry.Dispatch(context.Background(), &messagecommand.Request{
		Content: toggleCommand, Actor: command.Actor{UserID: "200"}, Responder: toggleResponder,
	})
	if err != nil || !handled || !strings.Contains(toggleResponder.content, "desativada") {
		t.Fatalf("toggle = %t, %v, %q", handled, err, toggleResponder.content)
	}

	listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
	board := testBoardMessage("board", "", "grid-cell")
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	loader := &fakeMessageLoader{messages: map[string]*discordgo.Message{"board": board}}
	listener.messages = loader
	manualResponder := &commandResponder{}
	handled, err = registry.Dispatch(context.Background(), &messagecommand.Request{
		Content: helperCommand, Actor: command.Actor{UserID: "200"},
		ChannelID: "300", ReplyToID: "board", Responder: manualResponder,
	})
	if err != nil || !handled || manualResponder.content != "" {
		t.Fatalf("manual = %t, %v, %q", handled, err, manualResponder.content)
	}
	call := receiveCall(t, messenger.sent)
	if call.sourceID != "board" {
		t.Fatalf("manual helper = %#v", call)
	}
	again := &commandResponder{}
	handled, err = registry.Dispatch(context.Background(), &messagecommand.Request{
		Content: helperCommand, Actor: command.Actor{UserID: "200"},
		ChannelID: "300", ReplyToID: "board", Responder: again,
	})
	if err != nil || !handled || !strings.Contains(again.content, "já está ativa") {
		t.Fatalf("second manual = %t, %v, %q", handled, err, again.content)
	}
}

func TestManualCommandValidatesReplyTarget(t *testing.T) {
	listener, _ := testListener(t)
	tests := []struct {
		name    string
		request *messagecommand.Request
		message *discordgo.Message
		want    string
	}{
		{"usage", &messagecommand.Request{Content: helperCommand + " extra"}, nil, "Responda"},
		{"not Mudae", &messagecommand.Request{Content: helperCommand, ChannelID: "300", ReplyToID: "target"}, &discordgo.Message{ID: "target", Author: &discordgo.User{ID: "other"}}, "não foi enviada"},
		{"not board", &messagecommand.Request{Content: helperCommand, ChannelID: "300", ReplyToID: "target"}, &discordgo.Message{ID: "target", Author: &discordgo.User{ID: "100"}}, "não contém"},
		{"oh", &messagecommand.Request{Content: helperCommand, ChannelID: "300", ReplyToID: "target"}, testBoardMessage("target", "$oh", "oh:cell"), "confirmar"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			responder := &commandResponder{}
			test.request.Actor.UserID = "200"
			test.request.Responder = responder
			listener.messages = &fakeMessageLoader{messages: map[string]*discordgo.Message{"target": test.message}}
			if err := listener.handleHelperCommand(context.Background(), test.request); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(responder.content, test.want) {
				t.Fatalf("response = %q", responder.content)
			}
		})
	}
}

func TestMessageCommandsPropagateDependencyErrors(t *testing.T) {
	listener, _ := testListener(t)
	want := errors.New("database unavailable")
	listener.preferences.(*fakePreferences).err = want
	request := &messagecommand.Request{
		Actor: command.Actor{UserID: "200"}, Responder: &commandResponder{},
	}
	if err := listener.handleToggleCommand(context.Background(), request); !errors.Is(err, want) {
		t.Fatalf("toggle error = %v", err)
	}
	listener.messages = &fakeMessageLoader{err: want}
	request.ReplyToID = "target"
	request.ChannelID = "300"
	if err := listener.handleHelperCommand(context.Background(), request); !errors.Is(err, want) {
		t.Fatalf("helper error = %v", err)
	}
}

func TestToggleCommandValidatesUsageAndCanReenableAutomation(t *testing.T) {
	listener, _ := testListener(t)
	usage := &commandResponder{}
	if err := listener.handleToggleCommand(context.Background(), &messagecommand.Request{
		Actor: command.Actor{UserID: "200"}, Arguments: []string{"extra"}, Responder: usage,
	}); err != nil || !strings.Contains(usage.content, "sem argumentos") {
		t.Fatalf("usage response = %q, %v", usage.content, err)
	}
	if err := listener.handleToggleCommand(context.Background(), &messagecommand.Request{
		Actor: command.Actor{UserID: "invalid"}, Responder: &commandResponder{},
	}); err == nil {
		t.Fatal("invalid user ID error = nil")
	}
	preferences := listener.preferences.(*fakePreferences)
	preferences.automatic = false
	responder := &commandResponder{}
	if err := listener.handleToggleCommand(context.Background(), &messagecommand.Request{
		Actor: command.Actor{UserID: "200"}, Responder: responder,
	}); err != nil || !strings.Contains(responder.content, "ativada") {
		t.Fatalf("enabled response = %q, %v", responder.content, err)
	}
}

type commandResponder struct {
	content string
}

func (r *commandResponder) Reply(content string) error {
	r.content = content
	return nil
}
