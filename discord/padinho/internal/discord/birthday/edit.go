package birthday

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	appbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/birthday"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const (
	editValueInputID = "value"
)

func (h Handler) Inspect(_ context.Context, request *discord.InteractionRequest) error {
	userID, err := strconv.ParseUint(string(request.Actor.UserID), 10, 64)
	if err != nil || userID == 0 {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayInvalidInteraction))
	}
	birthday, err := h.service.Birthday(userID)
	if errors.Is(err, appbirthday.ErrBirthdayNotFound) {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdaySelfNoRegistration))
	}
	if err != nil {
		return err
	}
	guild, err := h.guild(request.GuildID)
	if err != nil {
		return err
	}
	return request.Responder.Respond(inspectionResponse(birthday, guild))
}

// guild retrieves optional cached presentation data for an interaction guild.
func (h Handler) guild(guildID command.Snowflake) (*discordgo.Guild, error) {
	if h.guilds == nil {
		return nil, nil
	}
	return h.guilds.Guild(guildID)
}

func (h Handler) OpenDashboard(_ context.Context, request *discord.InteractionRequest) error {
	if !hasAdministratorPermission(request.Actor.Permissions) {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayAdministratorRequired))
	}
	return request.Responder.Respond(dashboardResponse(
		discordgo.InteractionResponseChannelMessageWithSource, 0, nil, "",
	))
}

func (h Handler) SelectDashboardUser(_ context.Context, request *discord.InteractionRequest) error {
	if !hasAdministratorPermission(request.Actor.Permissions) {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayAdministratorRequired))
	}
	userID, err := selectedUserID(request)
	if err != nil {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayInvalidInteraction))
	}
	birthday, err := h.service.Birthday(userID)
	if errors.Is(err, appbirthday.ErrBirthdayNotFound) {
		return request.Responder.Respond(dashboardResponse(
			discordgo.InteractionResponseUpdateMessage, userID, nil, ptbr.BirthdayNoRegistration,
		))
	}
	if err != nil {
		return err
	}
	return request.Responder.Respond(dashboardResponse(
		discordgo.InteractionResponseUpdateMessage, userID, birthday, "",
	))
}

func (h Handler) OpenEditModal(_ context.Context, request *discord.InteractionRequest) error {
	if !hasAdministratorPermission(request.Actor.Permissions) {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayAdministratorRequired))
	}
	field, userID, err := parseEditParameters(request.Parameters)
	if err != nil {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayInvalidInteraction))
	}
	birthday, err := h.service.Birthday(userID)
	if errors.Is(err, appbirthday.ErrBirthdayNotFound) {
		return request.Responder.Respond(dashboardResponse(
			discordgo.InteractionResponseUpdateMessage, userID, nil, ptbr.BirthdayNoRegistration,
		))
	}
	if err != nil {
		return err
	}
	return request.Responder.Respond(editModal(field, birthday))
}

func (h Handler) SubmitEdit(_ context.Context, request *discord.InteractionRequest) error {
	if !hasAdministratorPermission(request.Actor.Permissions) {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayAdministratorRequired))
	}
	field, userID, err := parseEditParameters(request.Parameters)
	if err != nil || request.Interaction == nil || request.Interaction.Type != discordgo.InteractionModalSubmit {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayInvalidInteraction))
	}
	value, exists := modalValues(request.Interaction.ModalSubmitData().Components)[editValueInputID]
	if !exists {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayInvalidInteraction))
	}
	input, err := updateInput(field, userID, value)
	if err != nil {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayInvalidDate))
	}
	birthday, err := h.service.Update(input)
	if errors.Is(err, appbirthday.ErrBirthdayNotFound) {
		return request.Responder.Respond(dashboardResponse(
			discordgo.InteractionResponseUpdateMessage, userID, nil, ptbr.BirthdayNoRegistration,
		))
	}
	if err != nil {
		message := validationMessage(err)
		if message != "" {
			return request.Responder.Respond(ephemeralMessage(message))
		}
		return err
	}
	return request.Responder.Respond(dashboardResponse(
		discordgo.InteractionResponseUpdateMessage, userID, birthday, ptbr.BirthdayEditSaved,
	))
}

func selectedUserID(request *discord.InteractionRequest) (uint64, error) {
	if request.Interaction == nil || request.Interaction.Type != discordgo.InteractionMessageComponent {
		return 0, errors.New("missing birthday dashboard select interaction")
	}
	values := request.Interaction.MessageComponentData().Values
	if len(values) != 1 {
		return 0, errors.New("invalid birthday dashboard selection")
	}
	userID, err := strconv.ParseUint(values[0], 10, 64)
	if err != nil || userID == 0 {
		return 0, errors.New("invalid birthday dashboard user")
	}
	return userID, nil
}

func parseEditParameters(parameters []string) (string, uint64, error) {
	if len(parameters) != 2 || !validEditField(parameters[0]) {
		return "", 0, errors.New("invalid birthday edit parameters")
	}
	userID, err := strconv.ParseUint(parameters[1], 10, 64)
	if err != nil || userID == 0 {
		return "", 0, errors.New("invalid birthday edit user")
	}
	return parameters[0], userID, nil
}

func validEditField(field string) bool {
	_, exists := birthdayField(field)
	return exists
}

func editModal(field string, birthday *entity.Birthday) *discordgo.InteractionResponse {
	definition, _ := birthdayField(field)
	value := editFieldValue(field, birthday)
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s:%s:%d", editSubmitRoute, field, birthday.UserID),
			Title:    fmt.Sprintf(ptbr.BirthdayEditModalTitle, definition.label),
			Components: []discordgo.MessageComponent{
				modalFieldLabel(definition, editValueInputID, value),
			},
		},
	}
}

// editFieldValue returns the stored value rendered in an edit modal control.
func editFieldValue(field string, birthday *entity.Birthday) string {
	switch field {
	case editFieldName:
		return birthday.Name
	case editFieldBirthday:
		return birthday.Birthday.Format(birthdayDateFormat)
	case editFieldTimeZone:
		return birthday.TimeZone
	default:
		return birthday.Message
	}
}

func updateInput(field string, userID uint64, value string) (appbirthday.UpdateInput, error) {
	input := appbirthday.UpdateInput{UserID: userID}
	switch field {
	case editFieldName:
		input.Name = &value
	case editFieldBirthday:
		birthday, err := time.Parse(birthdayDateFormat, value)
		if err != nil {
			return appbirthday.UpdateInput{}, err
		}
		input.Birthday = &birthday
	case editFieldTimeZone:
		input.TimeZone = &value
	case editFieldMessage:
		input.Message = &value
	default:
		return appbirthday.UpdateInput{}, errors.New("invalid birthday edit field")
	}
	return input, nil
}

func hasAdministratorPermission(permissions int64) bool {
	return permissions&discordgo.PermissionAdministrator != 0
}
