package birthday

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	appbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/birthday"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const (
	nameInputID          = "name"
	userInputID          = "user"
	birthdayInputID      = "birthday"
	timeZoneInputID      = "time_zone"
	messageInputID       = "message"
	maximumNameLength    = 100
	birthdayDateLength   = len("2006-01-02")
	maximumMessageLength = 1800
	brasiliaTimeZone     = "America/Sao_Paulo"
	amazonasTimeZone     = "America/Manaus"
	utcTimeZone          = "UTC"
)

func (h Handler) OpenModal(_ context.Context, request *discord.InteractionRequest) error {
	if !hasManageServerPermission(request.Actor.Permissions) {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayManageServerRequired))
	}
	return request.Responder.Respond(addBirthdayModal())
}

func addBirthdayModal() *discordgo.InteractionResponse {
	user := userLabel()
	name := inputLabel(
		nameInputID,
		ptbr.BirthdayNameLabel,
		ptbr.BirthdayNamePlaceholder,
		discordgo.TextInputShort,
		true,
		maximumNameLength,
	)
	birthday := inputLabel(
		birthdayInputID,
		ptbr.BirthdayDateLabel,
		ptbr.BirthdayDatePlaceholder,
		discordgo.TextInputShort,
		true,
		birthdayDateLength,
	)
	timeZone := timezoneLabel()
	message := inputLabel(
		messageInputID,
		ptbr.BirthdayMessageLabel,
		ptbr.BirthdayMessagePlaceholder,
		discordgo.TextInputParagraph,
		false,
		maximumMessageLength,
	)

	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: addBirthdayRoute,
			Title:    ptbr.BirthdayAddModalTitle,
			Components: []discordgo.MessageComponent{
				user,
				name,
				birthday,
				timeZone,
				message,
			},
		},
	}
}

func (h Handler) Submit(_ context.Context, request *discord.InteractionRequest) error {
	if !hasManageServerPermission(request.Actor.Permissions) {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayManageServerRequired))
	}
	values := modalValues(request.Interaction.ModalSubmitData().Components)
	userID, err := strconv.ParseUint(values[userInputID], 10, 64)
	if err != nil || userID == 0 {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayInvalidInteraction))
	}
	birthdayDate, err := time.Parse("2006-01-02", values[birthdayInputID])
	if err != nil {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayInvalidDate))
	}
	input := appbirthday.SaveInput{
		UserID:   userID,
		Name:     values[nameInputID],
		Birthday: birthdayDate,
		TimeZone: values[timeZoneInputID],
		Message:  values[messageInputID],
	}
	err = h.service.Save(input)
	if err == nil {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdaySaved))
	}
	message := validationMessage(err)
	if message == "" {
		return err
	}
	return request.Responder.Respond(ephemeralMessage(message))
}

// hasManageServerPermission accepts the explicit Manage Server permission and
// Administrator, whose effective permissions include every server action.
func hasManageServerPermission(permissions int64) bool {
	return permissions&(discordgo.PermissionManageGuild|discordgo.PermissionAdministrator) != 0
}

func inputLabel(
	customID string,
	label string,
	placeholder string,
	style discordgo.TextInputStyle,
	required bool,
	maximumLength int,
) discordgo.MessageComponent {
	requiredValue := required
	input := discordgo.TextInput{
		CustomID:    customID,
		Label:       label,
		Placeholder: placeholder,
		Style:       style,
		Required:    &requiredValue,
		MaxLength:   maximumLength,
	}
	return discordgo.Label{Label: label, Component: input}
}

func userLabel() discordgo.Label {
	required := true
	return discordgo.Label{
		Label:       ptbr.BirthdayUserLabel,
		Description: ptbr.BirthdayUserPlaceholder,
		Component: discordgo.SelectMenu{
			MenuType:    discordgo.UserSelectMenu,
			CustomID:    userInputID,
			Placeholder: ptbr.BirthdayUserPlaceholder,
			MinValues:   intPointer(1),
			MaxValues:   1,
			Required:    &required,
		},
	}
}

func timezoneLabel() discordgo.Label {
	required := true
	return discordgo.Label{
		Label:       ptbr.BirthdayTimeZoneLabel,
		Description: ptbr.BirthdayTimeZonePlaceholder,
		Component: discordgo.SelectMenu{
			MenuType:    discordgo.StringSelectMenu,
			CustomID:    timeZoneInputID,
			Placeholder: ptbr.BirthdayTimeZonePlaceholder,
			MinValues:   intPointer(1),
			MaxValues:   1,
			Required:    &required,
			Options: []discordgo.SelectMenuOption{
				{Label: ptbr.BirthdayTimeZoneBrasilia, Value: brasiliaTimeZone},
				{Label: ptbr.BirthdayTimeZoneAmazonas, Value: amazonasTimeZone},
				{Label: ptbr.BirthdayTimeZoneUTC, Value: utcTimeZone},
			},
		},
	}
}

func intPointer(value int) *int {
	return &value
}

func modalValues(components []discordgo.MessageComponent) map[string]string {
	values := make(map[string]string)
	for _, component := range components {
		collectModalValue(values, component)
	}
	return values
}

func collectModalValue(values map[string]string, component discordgo.MessageComponent) {
	switch input := component.(type) {
	case *discordgo.Label:
		collectModalValue(values, input.Component)
	case discordgo.Label:
		collectModalValue(values, input.Component)
	case *discordgo.ActionsRow:
		for _, child := range input.Components {
			collectModalValue(values, child)
		}
	case discordgo.ActionsRow:
		for _, child := range input.Components {
			collectModalValue(values, child)
		}
	case *discordgo.TextInput:
		values[input.CustomID] = input.Value
	case discordgo.TextInput:
		values[input.CustomID] = input.Value
	case *discordgo.SelectMenu:
		if len(input.Values) > 0 {
			values[input.CustomID] = input.Values[0]
		}
	case discordgo.SelectMenu:
		if len(input.Values) > 0 {
			values[input.CustomID] = input.Values[0]
		}
	}
}

func validationMessage(err error) string {
	switch {
	case errors.Is(err, appbirthday.ErrInvalidName):
		return ptbr.BirthdayInvalidName
	case errors.Is(err, appbirthday.ErrInvalidDate):
		return ptbr.BirthdayInvalidDate
	case errors.Is(err, appbirthday.ErrInvalidTimeZone):
		return ptbr.BirthdayInvalidTimeZone
	case errors.Is(err, appbirthday.ErrInvalidMessage):
		return ptbr.BirthdayInvalidMessage
	default:
		return ""
	}
}
