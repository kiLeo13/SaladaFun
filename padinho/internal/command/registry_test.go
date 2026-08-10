package command

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRegistryBuildsDefinitionsAndDispatchesInScopeOrder(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	var calls []string
	middleware := func(name string) Middleware {
		return func(next HandlerFunc) HandlerFunc {
			return func(ctx context.Context, request *CommandRequest) error {
				calls = append(calls, name)
				return next(ctx, request)
			}
		}
	}
	registry.Use(middleware("registry"))
	groups := registry.Group("groups", "Manage groups").Use(middleware("command"))
	members := groups.Group("members", "Manage members").Use(middleware("group"))
	members.Sub("add", "Add a member", func(_ context.Context, _ *CommandRequest) error {
		calls = append(calls, "handler")
		return nil
	}, StringOption("name", "Member name").Required()).Use(middleware("route"))

	if err := registry.Freeze(); err != nil {
		t.Fatalf("Freeze() error = %v", err)
	}
	if err := registry.Dispatch(context.Background(), &CommandRequest{Path: CommandPath{
		Command: "groups", Group: "members", Subcommand: "add",
	}}); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	wantCalls := []string{"registry", "command", "group", "route", "handler"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", calls, wantCalls)
	}

	definitions, err := registry.Definitions()
	if err != nil {
		t.Fatalf("Definitions() error = %v", err)
	}
	want := []Definition{{
		Name: "groups", Description: "Manage groups",
		Subcommands: []SubcommandDefinition{},
		Groups: []SubcommandGroupDefinition{{
			Name: "members", Description: "Manage members",
			Subcommands: []SubcommandDefinition{{
				Name: "add", Description: "Add a member",
				Options: []OptionDefinition{{
					Type: OptionTypeString, Name: "name", Description: "Member name", Required: true,
				}},
			}},
		}},
	}}
	if !reflect.DeepEqual(definitions, want) {
		t.Fatalf("Definitions() = %#v, want %#v", definitions, want)
	}

	definitions[0].Groups[0].Subcommands[0].Name = "changed"
	again, _ := registry.Definitions()
	if again[0].Groups[0].Subcommands[0].Name != "add" {
		t.Fatal("Definitions() exposed mutable registry state")
	}
}

func TestRegistrySupportsLeafAndDirectSubcommandHandlers(t *testing.T) {
	t.Parallel()

	target := &handlerService{}
	registry := NewRegistry()
	registry.Slash("ping", "Check the bot", target.Handle, BooleanOption("quiet", "Respond quietly"))
	registry.Group("admin", "Admin tools").Sub("ban", "Ban a user", target.Handle, UserOption("user", "User to ban").Required())
	if err := registry.Freeze(); err != nil {
		t.Fatalf("Freeze() error = %v", err)
	}
	for _, path := range []CommandPath{{Command: "ping"}, {Command: "admin", Subcommand: "ban"}} {
		if err := registry.Dispatch(context.Background(), &CommandRequest{Path: path}); err != nil {
			t.Fatalf("Dispatch(%v) error = %v", path, err)
		}
	}
	if target.calls != 2 {
		t.Fatalf("handler calls = %d, want 2", target.calls)
	}
	definitions, err := registry.Definitions()
	if err != nil || len(definitions) != 2 || len(definitions[1].Subcommands) != 1 {
		t.Fatalf("Definitions() = %#v, %v", definitions, err)
	}
}

func TestRegistryLifecycleErrors(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if _, err := registry.Definitions(); !errors.Is(err, ErrRegistryNotFrozen) {
		t.Fatalf("Definitions() error = %v", err)
	}
	if err := registry.Dispatch(context.Background(), &CommandRequest{}); !errors.Is(err, ErrRegistryNotFrozen) {
		t.Fatalf("Dispatch() before Freeze error = %v", err)
	}
	registry.Slash("ping", "Check the bot", func(context.Context, *CommandRequest) error { return nil })
	if err := registry.Freeze(); err != nil {
		t.Fatal(err)
	}
	if err := registry.Freeze(); err != nil {
		t.Fatalf("second Freeze() error = %v", err)
	}
	if err := registry.Dispatch(context.Background(), nil); !errors.Is(err, ErrNilRequest) {
		t.Fatalf("nil Dispatch() error = %v", err)
	}
	err := registry.Dispatch(context.Background(), &CommandRequest{Path: CommandPath{Command: "missing"}})
	if !errors.Is(err, ErrUnknownCommand) || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("unknown Dispatch() error = %v", err)
	}
	defer func() {
		if recovered := recover(); !errors.Is(asError(recovered), ErrRegistryFrozen) {
			t.Fatalf("mutation panic = %v", recovered)
		}
	}()
	registry.Use(func(next HandlerFunc) HandlerFunc { return next })
}

func TestRegistryRejectsMiddlewareThatReturnsNil(t *testing.T) {
	t.Parallel()

	builders := []func(*Registry){
		func(registry *Registry) { registry.Slash("ping", "Check", testHandler) },
		func(registry *Registry) { registry.Group("admin", "Admin").Sub("ban", "Ban", testHandler) },
		func(registry *Registry) {
			registry.Group("admin", "Admin").Group("users", "Users").Sub("add", "Add", testHandler)
		},
	}
	for index, build := range builders {
		registry := NewRegistry()
		registry.Use(func(HandlerFunc) HandlerFunc { return nil })
		build(registry)
		var validation *ValidationError
		if err := registry.Freeze(); !errors.As(err, &validation) || !strings.Contains(err.Error(), "nil handler") {
			t.Fatalf("case %d Freeze() error = %v", index, err)
		}
	}
}

func TestRegistryValidation(t *testing.T) {
	t.Parallel()

	tests := map[string]func(*Registry){
		"invalid metadata":    func(r *Registry) { r.Slash("UPPER", "", testHandler) },
		"nil leaf handler":    func(r *Registry) { r.Slash("ping", "Check", nil) },
		"empty command group": func(r *Registry) { r.Group("admin", "Admin") },
		"empty subgroup":      func(r *Registry) { r.Group("admin", "Admin").Group("users", "Users") },
		"too many root children": func(r *Registry) {
			group := r.Group("admin", "Admin")
			for index := 0; index <= maximumCommandOptions; index++ {
				group.Sub("s"+strings.Repeat("x", index), "Sub", testHandler)
			}
		},
		"too many grouped subcommands": func(r *Registry) {
			group := r.Group("admin", "Admin").Group("users", "Users")
			for index := 0; index <= maximumCommandOptions; index++ {
				group.Sub("s"+strings.Repeat("x", index), "Sub", testHandler)
			}
		},
		"duplicate root": func(r *Registry) { r.Slash("ping", "One", testHandler); r.Slash("ping", "Two", testHandler) },
		"duplicate direct child": func(r *Registry) {
			g := r.Group("admin", "Admin")
			g.Sub("ban", "One", testHandler)
			g.Sub("ban", "Two", testHandler)
		},
		"duplicate group child": func(r *Registry) {
			g := r.Group("admin", "Admin")
			g.Sub("users", "One", testHandler)
			g.Group("users", "Two").Sub("add", "Add", testHandler)
		},
		"duplicate grouped subcommand": func(r *Registry) {
			g := r.Group("admin", "Admin").Group("users", "Users")
			g.Sub("add", "One", testHandler)
			g.Sub("add", "Two", testHandler)
		},
		"nil registry middleware": func(r *Registry) { r.Use(nil); r.Slash("ping", "Check", testHandler) },
		"nil command middleware":  func(r *Registry) { r.Slash("ping", "Check", testHandler).Use(nil) },
		"nil group middleware":    func(r *Registry) { g := r.Group("admin", "Admin"); g.Use(nil); g.Sub("ban", "Ban", testHandler) },
		"nil subgroup middleware": func(r *Registry) {
			g := r.Group("admin", "Admin").Group("users", "Users")
			g.Use(nil)
			g.Sub("add", "Add", testHandler)
		},
		"nil route middleware": func(r *Registry) { r.Group("admin", "Admin").Sub("ban", "Ban", testHandler).Use(nil) },
		"invalid option":       func(r *Registry) { r.Slash("ping", "Check", testHandler, nil) },
		"duplicate option": func(r *Registry) {
			r.Slash("ping", "Check", testHandler, StringOption("value", "One"), StringOption("value", "Two"))
		},
		"required after optional": func(r *Registry) {
			r.Slash("ping", "Check", testHandler, StringOption("first", "First"), StringOption("second", "Second").Required())
		},
		"too many options": func(r *Registry) {
			options := make([]Option, maximumCommandOptions+1)
			for i := range options {
				options[i] = StringOption("a"+strings.Repeat("b", i), "Value")
			}
			r.Slash("ping", "Check", testHandler, options...)
		},
		"unsupported autocomplete": func(r *Registry) {
			option := BooleanOption("quiet", "Quiet")
			option.definition.Autocomplete = true
			r.Slash("ping", "Check", testHandler, option)
		},
	}
	for name, build := range tests {
		name, build := name, build
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			registry := NewRegistry()
			build(registry)
			var validation *ValidationError
			if err := registry.Freeze(); !errors.As(err, &validation) || len(validation.Problems) == 0 {
				t.Fatalf("Freeze() error = %v, want ValidationError", err)
			}
			if !strings.HasPrefix(validation.Error(), "invalid command registry:") {
				t.Fatalf("ValidationError.Error() = %q", validation.Error())
			}
		})
	}
}

func TestRejectionsAndMiddlewareShortCircuit(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	called := false
	registry.Use(func(HandlerFunc) HandlerFunc {
		return func(context.Context, *CommandRequest) error { return RejectForbidden("blocked") }
	})
	registry.Slash("ping", "Check", func(context.Context, *CommandRequest) error { called = true; return nil })
	if err := registry.Freeze(); err != nil {
		t.Fatal(err)
	}
	err := registry.Dispatch(context.Background(), &CommandRequest{Path: CommandPath{Command: "ping"}})
	rejection, ok := AsRejection(err)
	if !ok || rejection.Code != RejectionForbidden || rejection.Error() != "blocked" || called {
		t.Fatalf("rejection = %#v, ok = %v, called = %v", rejection, ok, called)
	}
	cooldown := RejectCooldown(time.Second)
	rejection, ok = AsRejection(cooldown)
	if !ok || rejection.Code != RejectionCooldown || rejection.RetryAfter != time.Second {
		t.Fatalf("cooldown rejection = %#v, ok = %v", rejection, ok)
	}
	if _, ok := AsRejection(errors.New("other")); ok {
		t.Fatal("plain error recognized as rejection")
	}
	if (&Rejection{Code: RejectionForbidden}).Error() != "command rejected: forbidden" {
		t.Fatal("unexpected default rejection text")
	}
}

func testHandler(context.Context, *CommandRequest) error { return nil }

type handlerService struct{ calls int }

func (s *handlerService) Handle(context.Context, *CommandRequest) error {
	s.calls++
	return nil
}

func asError(value any) error {
	err, _ := value.(error)
	return err
}
