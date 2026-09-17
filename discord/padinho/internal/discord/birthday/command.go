package birthday

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/enus"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const (
	monthOptionName    = "month"
	fullDateOptionName = "full-date"
)

var errInvalidMonth = errors.New("invalid birthday month")

// Handler owns the cohesive /birthdays command, button, and modal flow.
type Handler struct {
	service Service
	guilds  GuildLookup
	now     func() time.Time
}

func (h Handler) List(_ context.Context, request *command.CommandRequest) error {
	month, err := h.month(request)
	if err != nil {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayInvalidMonth))
	}
	fullDate, err := h.fullDate(request)
	if err != nil {
		return err
	}
	birthdays, err := h.service.Month(month)
	if err != nil {
		return err
	}
	next, err := h.service.Next(h.currentTime())
	if err != nil {
		return err
	}
	return request.Responder.Respond(pageResponse(
		discordgo.InteractionResponseChannelMessageWithSource,
		month,
		birthdays,
		next,
		fullDate,
	))
}

func (h Handler) month(request *command.CommandRequest) (time.Month, error) {
	value, err := request.Options.String(monthOptionName)
	if errors.Is(err, command.ErrOptionMissing) {
		return h.currentTime().Month(), nil
	}
	if err != nil {
		return 0, err
	}
	for month := time.January; month <= time.December; month++ {
		if value == enus.MonthNames[month] {
			return month, nil
		}
	}
	return 0, fmt.Errorf("%w: %s", errInvalidMonth, value)
}

func (h Handler) currentTime() time.Time {
	if h.now != nil {
		return h.now()
	}
	return time.Now()
}

// fullDate returns whether the command should include stored birth years.
func (h Handler) fullDate(request *command.CommandRequest) (bool, error) {
	value, err := request.Options.Boolean(fullDateOptionName)
	if errors.Is(err, command.ErrOptionMissing) {
		return false, nil
	}
	return value, err
}

func monthOption() *command.StringCommandOption {
	choices := make([]command.OptionChoice, 0, len(enus.MonthNames)-1)
	for month := time.January; month <= time.December; month++ {
		value := enus.MonthNames[month]
		choices = append(choices, command.OptionChoice{
			Name: locale.Capitalize(value), Value: value,
		})
	}
	return command.StringOption(monthOptionName, enus.BirthdayMonthOptionDescription).Choices(choices...)
}

// fullDateOption declares the optional year-display toggle.
func fullDateOption() *command.BooleanCommandOption {
	return command.BooleanOption(fullDateOptionName, enus.BirthdayFullDateOptionDescription)
}
