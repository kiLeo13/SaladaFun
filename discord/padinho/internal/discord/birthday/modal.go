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
	nameInputID           = "name"
	birthdayInputID       = "birthday"
	timeZoneInputID       = "time_zone"
	messageInputID        = "message"
	maximumNameLength     = 100
	birthdayDateLength    = len("2006-01-02")
	maximumTimeZoneLength = 255
	maximumMessageLength  = 1800
)

func (h Handler) OpenModal(_ context.Context, request *discord.InteractionRequest) error {
	return request.Responder.Respond(addBirthdayModal())
}

func addBirthdayModal() *discordgo.InteractionResponse {
	name := inputRow(
		nameInputID,
		ptbr.BirthdayNameLabel,
		ptbr.BirthdayNamePlaceholder,
		discordgo.TextInputShort,
		true,
		maximumNameLength,
	)
	birthday := inputRow(
		birthdayInputID,
		ptbr.BirthdayDateLabel,
		ptbr.BirthdayDatePlaceholder,
		discordgo.TextInputShort,
		true,
		birthdayDateLength,
	)
	timeZone := inputRow(
		timeZoneInputID,
		ptbr.BirthdayTimeZoneLabel,
		ptbr.BirthdayTimeZonePlaceholder,
		discordgo.TextInputShort,
		true,
		maximumTimeZoneLength,
	)
	message := inputRow(
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
				name,
				birthday,
				timeZone,
				message,
			},
		},
	}
}

func (h Handler) Submit(_ context.Context, request *discord.InteractionRequest) error {
	values := modalValues(request.Interaction.ModalSubmitData().Components)
	userID, err := strconv.ParseUint(string(request.Actor.UserID), 10, 64)
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

func inputRow(
	customID string,
	label string,
	placeholder string,
	style discordgo.TextInputStyle,
	required bool,
	maximumLength int,
) discordgo.ActionsRow {
	input := discordgo.TextInput{
		CustomID:    customID,
		Label:       label,
		Placeholder: placeholder,
		Style:       style,
		Required:    required,
		MaxLength:   maximumLength,
	}
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{input}}
}

func modalValues(components []discordgo.MessageComponent) map[string]string {
	values := make(map[string]string)
	for _, component := range components {
		row, ok := component.(*discordgo.ActionsRow)
		if !ok {
			if value, valueOK := component.(discordgo.ActionsRow); valueOK {
				row = &value
			} else {
				continue
			}
		}
		for _, child := range row.Components {
			switch input := child.(type) {
			case *discordgo.TextInput:
				values[input.CustomID] = input.Value
			case discordgo.TextInput:
				values[input.CustomID] = input.Value
			}
		}
	}
	return values
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
