package accounttree

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	appaccounttree "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/accounttree"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

func TestCommandUsesFirstArgumentAndRendersRootTree(t *testing.T) {
	root := &appaccounttree.Node{UserID: 100}
	root.Children = []*appaccounttree.Node{{UserID: 200}, {UserID: 300}}
	service := &fakeService{tree: &appaccounttree.Tree{Root: root, Count: 3}}
	members := fakeMembers{names: map[command.Snowflake]string{"200": "Lucas", "100": "Mike", "300": "Amanda"}}
	responder := &fakeResponder{}
	registry := messagecommand.NewRegistry()
	Register(registry, service, members)
	if err := registry.Freeze(); err != nil {
		t.Fatal(err)
	}
	handled, err := registry.Dispatch(context.Background(), &messagecommand.Request{
		Content: "!childrentree <@200> ignored arguments", GuildID: "guild", Actor: command.Actor{UserID: "999"}, Responder: responder,
	})
	if err != nil || !handled || service.requestedID != 200 || len(responder.components) != 1 {
		t.Fatalf("Dispatch() = %t, %v, %#v", handled, err, responder)
	}
	container, ok := responder.components[0].(discordgo.Container)
	if !ok || container.AccentColor != nil || len(container.Components) != 2 {
		t.Fatalf("container = %#v", responder.components)
	}
	title := container.Components[0].(discordgo.TextDisplay).Content
	body := container.Components[1].(discordgo.TextDisplay).Content
	if title != "## Árvore de <@100>" || !strings.Contains(body, "Mike\n|__ Amanda\n|__ Lucas") || !strings.Contains(body, "-# 3 contas totais") {
		t.Fatalf("rendered content = %q / %q", title, body)
	}
}

func TestCommandDefaultsToActorAndReportsInvalidOrMissingUsers(t *testing.T) {
	tree := &appaccounttree.Tree{Root: &appaccounttree.Node{UserID: 10}, Count: 1}
	service := &fakeService{tree: tree}
	members := fakeMembers{names: map[command.Snowflake]string{"10": "Mike"}}
	responder := &fakeResponder{}
	if err := handle(context.Background(), &messagecommand.Request{
		GuildID: "guild", Actor: command.Actor{UserID: "10"}, Responder: responder,
	}, service, members); err != nil || service.requestedID != 10 || len(responder.components) != 1 {
		t.Fatalf("default handle() = %v, %#v", err, responder)
	}
	responder = &fakeResponder{}
	if err := handle(context.Background(), &messagecommand.Request{
		Arguments: []string{"no-id"}, Responder: responder,
	}, service, members); err != nil || responder.text != ptbr.AccountTreeInvalidID {
		t.Fatalf("invalid ID handle() = %v, %#v", err, responder)
	}
	responder = &fakeResponder{}
	if err := handle(context.Background(), &messagecommand.Request{
		Arguments: []string{"20"}, GuildID: "guild", Responder: responder,
	}, service, members); err != nil || responder.text != ptbr.AccountTreeUserNotFound {
		t.Fatalf("missing user handle() = %v, %#v", err, responder)
	}
}

func TestCommandPropagatesDependenciesAndRequiresComponentResponder(t *testing.T) {
	want := errors.New("database unavailable")
	request := &messagecommand.Request{GuildID: "guild", Actor: command.Actor{UserID: "10"}, Responder: &fakeResponder{}}
	if err := handle(context.Background(), request, &fakeService{err: want}, fakeMembers{names: map[command.Snowflake]string{"10": "Mike"}}); !errors.Is(err, want) {
		t.Fatalf("service failure = %v", err)
	}
	if err := handle(context.Background(), &messagecommand.Request{
		GuildID: "guild", Actor: command.Actor{UserID: "10"}, Responder: replyOnlyResponder{},
	}, &fakeService{tree: &appaccounttree.Tree{Root: &appaccounttree.Node{UserID: 10}, Count: 1}}, fakeMembers{names: map[command.Snowflake]string{"10": "Mike"}}); err == nil {
		t.Fatal("unsupported responder error = nil")
	}
}

type fakeService struct {
	tree        *appaccounttree.Tree
	err         error
	requestedID uint64
}

// Tree records its argument and returns the configured result.
func (s *fakeService) Tree(userID uint64) (*appaccounttree.Tree, error) {
	s.requestedID = userID
	return s.tree, s.err
}

type fakeMembers struct {
	names map[command.Snowflake]string
	err   error
}

// GuildMemberDisplayName returns the configured name when present.
func (m fakeMembers) GuildMemberDisplayName(_ command.Snowflake, userID command.Snowflake) (string, bool, error) {
	if m.err != nil {
		return "", false, m.err
	}
	name, found := m.names[userID]
	return name, found, nil
}

type fakeResponder struct {
	text       string
	components []discordgo.MessageComponent
}

// Reply records a plain localized rejection.
func (r *fakeResponder) Reply(content string) error {
	r.text = content
	return nil
}

// ReplyWithComponents records one Components V2 response.
func (r *fakeResponder) ReplyWithComponents(components []discordgo.MessageComponent) error {
	r.components = components
	return nil
}

type replyOnlyResponder struct{}

// Reply satisfies the basic message-command responder contract.
func (replyOnlyResponder) Reply(string) error { return nil }
