package birthday

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	appbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/birthday"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

func (h Handler) OpenModal(_ context.Context, request *discord.InteractionRequest) error {
	if !hasManageServerPermission(request.Actor.Permissions) {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayManageServerRequired))
	}
	return request.Responder.Respond(addBirthdayModal())
}

func addBirthdayModal() *discordgo.InteractionResponse {
	definitions := birthdayFieldDefinitions()
	components := make([]discordgo.MessageComponent, 0, len(definitions)+1)
	components = append(components, userLabel())
	for _, definition := range definitions {
		components = append(components, modalFieldLabel(definition, definition.id, ""))
	}

	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   addBirthdayRoute,
			Title:      ptbr.BirthdayAddModalTitle,
			Components: components,
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
	birthdayDate, err := time.Parse(birthdayDateFormat, values[birthdayInputID])
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
		return request.Responder.Respond(savedMessage(userID, string(request.Actor.UserID)))
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

func userLabel() discordgo.Label {
	required := true
	return discordgo.Label{
		Label: ptbr.BirthdayUserLabel,
		Component: discordgo.SelectMenu{
			MenuType:    discordgo.UserSelectMenu,
			CustomID:    userInputID,
			Placeholder: ptbr.BirthdayUserPlaceholder,
			MinValues:   new(1),
			MaxValues:   1,
			Required:    &required,
		},
	}
}

func savedMessage(userID uint64, actorUserID string) *discordgo.InteractionResponse {
	if strconv.FormatUint(userID, 10) == actorUserID {
		return ephemeralMessage(ptbr.BirthdaySaved)
	}
	response := ephemeralMessage(fmt.Sprintf(ptbr.BirthdaySavedForUser, userID))
	response.Data.AllowedMentions = &discordgo.MessageAllowedMentions{Users: []string{strconv.FormatUint(userID, 10)}}
	return response
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
