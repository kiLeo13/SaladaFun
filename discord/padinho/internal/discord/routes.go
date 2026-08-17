package discord

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
)

var (
	ErrRoutesFrozen    = errors.New("Discord routes are frozen")
	ErrRoutesNotFrozen = errors.New("Discord routes are not frozen")
	ErrUnknownRoute    = errors.New("unknown Discord interaction route")
)

// InteractionRequest contains the typed common data passed to component and
// modal handlers after Routes has decoded a Discord custom_id.
type InteractionRequest struct {
	// Actor identifies the user and effective guild permissions for the request.
	Actor command.Actor
	// GuildID is the guild where the interaction occurred; it is empty in DMs.
	GuildID command.Snowflake
	// ChannelID is the channel where the interaction occurred.
	ChannelID command.Snowflake
	// Parameters are the custom_id segments after the registered route. For
	// example, "birthdays.page:next:1" produces ["next", "1"]. They are
	// untrusted client input and every handler must validate their shape/value.
	Parameters []string
	// Responder sends the one initial response for this interaction.
	Responder command.Responder
	// Interaction is the original native Discord payload for handlers that need
	// interaction-specific data, such as modal text-input values.
	Interaction *discordgo.InteractionCreate
}

// InteractionHandler handles a routed message component or modal submission.
type InteractionHandler func(context.Context, *InteractionRequest) error

// Routes owns Padinho's one command registry and its component/modal routes.
type Routes struct {
	mu         sync.RWMutex
	commands   *command.Registry
	components map[string]InteractionHandler
	modals     map[string]InteractionHandler
	frozen     bool
}

func NewRoutes() *Routes {
	return &Routes{
		commands:   command.NewRegistry(),
		components: make(map[string]InteractionHandler),
		modals:     make(map[string]InteractionHandler),
	}
}

// Commands returns the unique slash-command registry.
func (r *Routes) Commands() *command.Registry {
	return r.commands
}

// Component registers one stable component route. Runtime parameters follow
// the route in custom_id, separated by colons.
func (r *Routes) Component(route string, handler InteractionHandler) {
	r.register(r.components, route, handler)
}

// Modal registers one modal custom_id.
func (r *Routes) Modal(route string, handler InteractionHandler) {
	r.register(r.modals, route, handler)
}

func (r *Routes) register(routes map[string]InteractionHandler, route string, handler InteractionHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		panic(ErrRoutesFrozen)
	}
	if route == "" || strings.Contains(route, ":") {
		panic(fmt.Errorf("invalid Discord interaction route %q", route))
	}
	if handler == nil {
		panic(fmt.Errorf("Discord interaction route %q has no handler", route))
	}
	if _, exists := routes[route]; exists {
		panic(fmt.Errorf("duplicate Discord interaction route %q", route))
	}
	routes[route] = handler
}

// Freeze validates slash commands and makes every interaction route immutable.
func (r *Routes) Freeze() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		return nil
	}
	if err := r.commands.Freeze(); err != nil {
		return err
	}
	r.frozen = true
	return nil
}

func (r *Routes) dispatch(
	ctx context.Context,
	interaction *discordgo.InteractionCreate,
	responder command.Responder,
) (bool, error) {
	r.mu.RLock()
	if !r.frozen {
		r.mu.RUnlock()
		return true, ErrRoutesNotFrozen
	}
	r.mu.RUnlock()

	switch interaction.Type {
	case discordgo.InteractionApplicationCommand:
		request, err := mapRequest(interaction, responder)
		if err != nil {
			return true, err
		}
		return true, r.commands.Dispatch(ctx, request)
	case discordgo.InteractionMessageComponent:
		data := interaction.MessageComponentData()
		return true, r.dispatchRoute(ctx, r.components, data.CustomID, interaction, responder)
	case discordgo.InteractionModalSubmit:
		data := interaction.ModalSubmitData()
		return true, r.dispatchRoute(ctx, r.modals, data.CustomID, interaction, responder)
	default:
		return false, nil
	}
}

func (r *Routes) dispatchRoute(
	ctx context.Context,
	routes map[string]InteractionHandler,
	customID string,
	interaction *discordgo.InteractionCreate,
	responder command.Responder,
) error {
	segments := strings.Split(customID, ":")
	r.mu.RLock()
	handler, exists := routes[segments[0]]
	r.mu.RUnlock()
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownRoute, segments[0])
	}
	return handler(ctx, &InteractionRequest{
		Actor: actor(interaction), GuildID: command.Snowflake(interaction.GuildID),
		ChannelID:  command.Snowflake(interaction.ChannelID),
		Parameters: append([]string(nil), segments[1:]...), Responder: responder,
		Interaction: interaction,
	})
}

func actor(interaction *discordgo.InteractionCreate) command.Actor {
	result := command.Actor{}
	if interaction.Member != nil {
		if interaction.Member.User != nil {
			result.UserID = command.Snowflake(interaction.Member.User.ID)
		}
		result.RoleIDs = make([]command.Snowflake, len(interaction.Member.Roles))
		for index, role := range interaction.Member.Roles {
			result.RoleIDs[index] = command.Snowflake(role)
		}
		result.Permissions = interaction.Member.Permissions
	} else if interaction.User != nil {
		result.UserID = command.Snowflake(interaction.User.ID)
	}
	return result
}
