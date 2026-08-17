package command

import (
	"errors"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
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
	// UserID is the Discord user who initiated the interaction.
	UserID Snowflake
	// RoleIDs are the guild roles Discord reported for the user.
	RoleIDs []Snowflake
	// Permissions is the effective guild permission bitmask for the user in the
	// interaction context. The transport layer obtains it from Discord; feature
	// handlers compare it with Discord permission constants when needed.
	Permissions int64
}

// Responder sends one native Discord initial interaction response. It remains
// bound to the originating interaction so handlers cannot accidentally answer
// a different request or send two initial responses.
type Responder interface {
	Respond(*discordgo.InteractionResponse) error
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
	// Path identifies the slash command, group, and subcommand being executed.
	Path CommandPath
	// Actor identifies the user and guild capabilities associated with it.
	Actor Actor
	// GuildID is the guild where the command was invoked; it is empty in DMs.
	GuildID Snowflake
	// ChannelID is the channel where the command was invoked.
	ChannelID Snowflake
	// Options contains the command's validated, typed option values.
	Options OptionValues
	// Responder sends the one initial response for this interaction.
	Responder Responder
	// RequestID is Discord's unique interaction ID, useful for correlation logs.
	RequestID string
	// ReceivedAt is the UTC time at which Padinho mapped the Discord request.
	ReceivedAt time.Time
}
