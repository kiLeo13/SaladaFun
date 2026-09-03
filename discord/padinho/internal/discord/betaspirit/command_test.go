package betaspirit

import (
	"context"
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

func TestCommandRepliesWithConfiguredURL(t *testing.T) {
	registry := messagecommand.NewRegistry()
	Register(registry, "https://www.youtube.com/watch?v=TI0f3xA5_gg")
	if err := registry.Freeze(); err != nil {
		t.Fatal(err)
	}
	responder := &fakeResponder{}
	handled, err := registry.Dispatch(context.Background(), &messagecommand.Request{
		Content: command, Responder: responder,
	})
	if err != nil || !handled || responder.content != "https://www.youtube.com/watch?v=TI0f3xA5_gg" {
		t.Fatalf("Dispatch() = %t, %v, %#v", handled, err, responder)
	}
}

type fakeResponder struct {
	content string
}

// Reply records the sent message.
func (r *fakeResponder) Reply(content string) error {
	r.content = content
	return nil
}
