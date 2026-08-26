// Package messagecommand declares and dispatches prefix-agnostic Discord message commands.
package messagecommand

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
)

var (
	// ErrRegistryFrozen rejects message-command registration after startup.
	ErrRegistryFrozen = errors.New("message command registry is frozen")
	// ErrRegistryNotFrozen rejects dispatch before startup validation completes.
	ErrRegistryNotFrozen = errors.New("message command registry is not frozen")
	// ErrNilRequest rejects an absent message-command request.
	ErrNilRequest = errors.New("message command request is nil")
)

// Responder sends a safe reply bound to the invoking Discord message.
type Responder interface {
	Reply(content string) error
}

// Request contains normalized data for one registered message command.
type Request struct {
	Actor        command.Actor
	GuildID      command.Snowflake
	ChannelID    command.Snowflake
	MessageID    command.Snowflake
	ReplyToID    command.Snowflake
	Trigger      string
	Content      string
	RawArguments string
	Arguments    []string
	Responder    Responder
	ReceivedAt   time.Time
}

// HandlerFunc executes one message command.
type HandlerFunc func(context.Context, *Request) error

type declaration struct {
	trigger string
	handler HandlerFunc
}

// Registry owns startup declarations and exact first-token dispatch.
type Registry struct {
	mu           sync.RWMutex
	frozen       bool
	declarations []declaration
	dispatch     map[string]HandlerFunc
}

// NewRegistry creates an empty message-command registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Command registers a complete literal trigger, including its prefix.
func (r *Registry) Command(trigger string, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		panic(ErrRegistryFrozen)
	}
	r.declarations = append(r.declarations, declaration{trigger: trigger, handler: handler})
}

// Freeze validates declarations and builds the immutable dispatch table.
func (r *Registry) Freeze() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		return nil
	}
	dispatch := make(map[string]HandlerFunc, len(r.declarations))
	for _, command := range r.declarations {
		trigger := strings.TrimSpace(command.trigger)
		if trigger == "" || trigger != command.trigger || len(strings.Fields(trigger)) != 1 {
			return fmt.Errorf("invalid message command trigger %q", command.trigger)
		}
		if command.handler == nil {
			return fmt.Errorf("message command %q has no handler", trigger)
		}
		key := strings.ToLower(trigger)
		if _, duplicate := dispatch[key]; duplicate {
			return fmt.Errorf("duplicate message command %q", trigger)
		}
		dispatch[key] = command.handler
	}
	r.dispatch = dispatch
	r.frozen = true
	return nil
}

// Dispatch parses content with strings.Fields and invokes an exact trigger match.
func (r *Registry) Dispatch(ctx context.Context, request *Request) (bool, error) {
	if request == nil {
		return false, ErrNilRequest
	}
	r.mu.RLock()
	if !r.frozen {
		r.mu.RUnlock()
		return false, ErrRegistryNotFrozen
	}
	fields := strings.Fields(request.Content)
	if len(fields) == 0 {
		r.mu.RUnlock()
		return false, nil
	}
	handler, exists := r.dispatch[strings.ToLower(fields[0])]
	r.mu.RUnlock()
	if !exists {
		return false, nil
	}

	mapped := *request
	mapped.Trigger = fields[0]
	mapped.RawArguments = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(request.Content), fields[0]))
	mapped.Arguments = append([]string(nil), fields[1:]...)
	return true, handler(ctx, &mapped)
}
