package birthday

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const birthdayAccentColor = 0x65A30D

func pageResponse(
	responseType discordgo.InteractionResponseType,
	month time.Month,
	birthdays []*entity.Birthday,
	ownerID string,
) *discordgo.InteractionResponse {
	content := pageContent(month, birthdays)
	container := pageContainer(content)
	actions := pageActions(month, ownerID)

	return &discordgo.InteractionResponse{
		Type: responseType,
		Data: &discordgo.InteractionResponseData{
			Flags:           discordgo.MessageFlagsIsComponentsV2,
			AllowedMentions: &discordgo.MessageAllowedMentions{},
			Components:      []discordgo.MessageComponent{container, actions},
		},
	}
}

func pageContainer(content string) discordgo.Container {
	accent := birthdayAccentColor
	return discordgo.Container{
		AccentColor: &accent,
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: content},
		},
	}
}

func pageActions(month time.Month, ownerID string) discordgo.ActionsRow {
	previous := pageButton(
		"⬅️",
		"previous",
		month,
		ownerID,
		month == time.January,
	)
	next := pageButton(
		"➡️",
		"next",
		month,
		ownerID,
		month == time.December,
	)

	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		previous,
		next,
		addButton(),
	}}
}

func pageContent(month time.Month, birthdays []*entity.Birthday) string {
	var content strings.Builder
	fmt.Fprintf(&content, "## "+ptbr.BirthdayTitle, ptbr.MonthNames[month])
	if len(birthdays) == 0 {
		content.WriteString("\n\n")
		content.WriteString(ptbr.BirthdayEmptyMonth)
		return content.String()
	}
	for _, birthday := range birthdays {
		content.WriteString("\n\n")
		fmt.Fprintf(
			&content,
			ptbr.BirthdayEntry,
			birthday.Birthday.Day(),
			birthday.Birthday.Month(),
			escapeMarkdown(birthday.Name),
		)
	}
	return content.String()
}

func escapeMarkdown(value string) string {
	return strings.NewReplacer(
		"\\", "\\\\", "*", "\\*", "_", "\\_", "~", "\\~",
		"`", "\\`", ">", "\\>", "|", "\\|",
	).Replace(value)
}

func pageButton(
	emoji string,
	direction string,
	month time.Month,
	ownerID string,
	disabled bool,
) discordgo.Button {
	return discordgo.Button{
		Style: discordgo.SecondaryButton, Disabled: disabled,
		Emoji:    &discordgo.ComponentEmoji{Name: emoji},
		CustomID: fmt.Sprintf("%s:%s:%d:%s", pageRoute, direction, month, ownerID),
	}
}

func addButton() discordgo.Button {
	return discordgo.Button{
		Style:    discordgo.SuccessButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "➕"},
		CustomID: addBirthdayRoute,
	}
}

func parsePage(parameters []string) (string, time.Month, string, error) {
	if len(parameters) != 3 || parameters[0] != "previous" && parameters[0] != "next" {
		return "", 0, "", fmt.Errorf("invalid birthday page parameters")
	}
	month, err := strconv.Atoi(parameters[1])
	if err != nil || month < int(time.January) || month > int(time.December) {
		return "", 0, "", fmt.Errorf("invalid birthday page month")
	}
	if parameters[2] == "" {
		return "", 0, "", fmt.Errorf("birthday page owner is empty")
	}
	return parameters[0], time.Month(month), parameters[2], nil
}
