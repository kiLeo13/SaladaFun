package messagecommand

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestRegistryDispatchesCompleteCaseInsensitiveTriggers(t *testing.T) {
	registry := NewRegistry()
	var received *Request
	registry.Command("!ochelper", func(_ context.Context, request *Request) error {
		received = request
		return nil
	})
	if err := registry.Freeze(); err != nil {
		t.Fatal(err)
	}
	request := &Request{Content: "  !OCHELPER   one\ttwo three  "}
	handled, err := registry.Dispatch(context.Background(), request)
	if err != nil || !handled || received == nil {
		t.Fatalf("Dispatch() = %t, %v; request = %#v", handled, err, received)
	}
	if received.Trigger != "!OCHELPER" || received.RawArguments != "one\ttwo three" ||
		!reflect.DeepEqual(received.Arguments, []string{"one", "two", "three"}) {
		t.Fatalf("mapped request = %#v", received)
	}
	if request.Trigger != "" || request.Arguments != nil {
		t.Fatalf("source request mutated = %#v", request)
	}

	for _, content := range []string{"", "   ", "!ochelper-extra", "hello !ochelper"} {
		handled, err = registry.Dispatch(context.Background(), &Request{Content: content})
		if err != nil || handled {
			t.Fatalf("Dispatch(%q) = %t, %v", content, handled, err)
		}
	}
}

func TestRegistryValidatesLifecycleAndDeclarations(t *testing.T) {
	tests := map[string]func(*Registry){
		"empty": func(registry *Registry) { registry.Command("", func(context.Context, *Request) error { return nil }) },
		"whitespace": func(registry *Registry) {
			registry.Command("!bad command", func(context.Context, *Request) error { return nil })
		},
		"nil": func(registry *Registry) { registry.Command("!nil", nil) },
		"duplicate": func(registry *Registry) {
			registry.Command("!same", func(context.Context, *Request) error { return nil })
			registry.Command("!SAME", func(context.Context, *Request) error { return nil })
		},
	}
	for name, configure := range tests {
		t.Run(name, func(t *testing.T) {
			registry := NewRegistry()
			configure(registry)
			if err := registry.Freeze(); err == nil {
				t.Fatal("Freeze() error = nil")
			}
		})
	}

	registry := NewRegistry()
	if handled, err := registry.Dispatch(context.Background(), &Request{Content: "!test"}); handled || !errors.Is(err, ErrRegistryNotFrozen) {
		t.Fatalf("unfrozen Dispatch() = %t, %v", handled, err)
	}
	registry.Command("!test", func(context.Context, *Request) error { return nil })
	if err := registry.Freeze(); err != nil {
		t.Fatal(err)
	}
	if err := registry.Freeze(); err != nil {
		t.Fatal(err)
	}
	if handled, err := registry.Dispatch(context.Background(), nil); handled || !errors.Is(err, ErrNilRequest) {
		t.Fatalf("nil Dispatch() = %t, %v", handled, err)
	}
	defer func() {
		if recovered := recover(); !errors.Is(recovered.(error), ErrRegistryFrozen) {
			t.Fatalf("panic = %v", recovered)
		}
	}()
	registry.Command("!late", func(context.Context, *Request) error { return nil })
}
