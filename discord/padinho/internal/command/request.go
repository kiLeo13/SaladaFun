package command

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Snowflake is a Discord entity identifier. An empty Snowflake represents an
// entity that is not present, such as a guild for a direct-message command.
type Snowflake string

// CommandPath identifies a registered command route.
type CommandPath struct {
	Command    string
	Group      string
	Subcommand string
}

// Segments returns the non-empty command path components in Discord order.
func (p CommandPath) Segments() []string {
	segments := make([]string, 0, 3)
	for _, segment := range [...]string{p.Command, p.Group, p.Subcommand} {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	return segments
}

func (p CommandPath) key() string {
	segments := p.Segments()
	if len(segments) == 0 {
		return ""
	}

	key := segments[0]
	for _, segment := range segments[1:] {
		key += "\x00" + segment
	}
	return key
}

// Actor identifies the Discord user invoking a command and their known roles.
type Actor struct {
	UserID  Snowflake
	RoleIDs []Snowflake
}

// Response is a framework-neutral Discord response.
type Response struct {
	Content   string
	Ephemeral bool
}

// Responder sends the initial response for a command request.
type Responder interface {
	Respond(context.Context, Response) error
}

var (
	ErrOptionMissing = errors.New("command option is missing")
	ErrOptionType    = errors.New("command option has an unexpected type")
)

// OptionValues stores already-normalized command option values.
type OptionValues struct {
	values map[string]any
}

// NewOptionValues copies values into an immutable command option collection.
func NewOptionValues(values map[string]any) OptionValues {
	result := OptionValues{values: make(map[string]any, len(values))}
	for name, value := range values {
		result.values[name] = value
	}
	return result
}

// String returns a string option.
func (v OptionValues) String(name string) (string, error) {
	value, ok := v.values[name]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrOptionMissing, name)
	}
	typed, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s is %T", ErrOptionType, name, value)
	}
	return typed, nil
}

// Integer returns an integer option.
func (v OptionValues) Integer(name string) (int64, error) {
	value, ok := v.values[name]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrOptionMissing, name)
	}
	typed, ok := value.(int64)
	if !ok {
		return 0, fmt.Errorf("%w: %s is %T", ErrOptionType, name, value)
	}
	return typed, nil
}

// Boolean returns a Boolean option.
func (v OptionValues) Boolean(name string) (bool, error) {
	value, ok := v.values[name]
	if !ok {
		return false, fmt.Errorf("%w: %s", ErrOptionMissing, name)
	}
	typed, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%w: %s is %T", ErrOptionType, name, value)
	}
	return typed, nil
}

// Snowflake returns an entity option represented by a Discord Snowflake.
func (v OptionValues) Snowflake(name string) (Snowflake, error) {
	value, ok := v.values[name]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrOptionMissing, name)
	}
	typed, ok := value.(Snowflake)
	if !ok {
		return "", fmt.Errorf("%w: %s is %T", ErrOptionType, name, value)
	}
	return typed, nil
}

// CommandRequest contains all typed application data needed to execute a
// command. context.Context is reserved for cancellation and deadlines.
type CommandRequest struct {
	Path       CommandPath
	Actor      Actor
	GuildID    Snowflake
	ChannelID  Snowflake
	Options    OptionValues
	Responder  Responder
	RequestID  string
	ReceivedAt time.Time
}
