package birthday

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const emptyDashboardValue = "—"

func inspectionResponse(birthday *entity.Birthday) *discordgo.InteractionResponse {
	accent := birthdayAccentColor
	message := birthday.Message
	if message == "" {
		message = ptbr.BirthdayDefaultMessageValue
	}
	container := discordgo.Container{
		AccentColor: &accent,
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: "## " + ptbr.BirthdayInspectTitle},
			discordgo.Separator{},
			discordgo.TextDisplay{Content: fmt.Sprintf(
				"**%s**\n`%d`\n\n**%s**\n%s\n\n**%s**\n`%s`\n\n**%s**\n`%s`\n\n**%s**\n%s",
				ptbr.BirthdayUserIDLabel,
				birthday.UserID,
				ptbr.BirthdayNameLabel,
				escapeDisplayValue(birthday.Name),
				ptbr.BirthdayDateLabel,
				birthday.Birthday.Format(birthdayDateFormat),
				ptbr.BirthdayTimeZoneLabel,
				birthday.TimeZone,
				ptbr.BirthdayMessageLabel,
				escapeDisplayValue(message),
			)},
		},
	}
	return componentResponse(discordgo.InteractionResponseChannelMessageWithSource, true, container)
}

func dashboardResponse(
	responseType discordgo.InteractionResponseType,
	selectedUserID uint64,
	birthday *entity.Birthday,
	status string,
) *discordgo.InteractionResponse {
	accent := birthdayAccentColor
	components := []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: "## " + ptbr.BirthdayEditDashboardTitle},
		discordgo.Separator{},
	}
	if status != "" {
		components = append(components, discordgo.TextDisplay{Content: status})
	}
	components = append(components, dashboardFields(birthday)...)
	container := discordgo.Container{AccentColor: &accent, Components: components}
	response := componentResponse(responseType, responseType == discordgo.InteractionResponseChannelMessageWithSource, container, dashboardSelect(selectedUserID))
	return response
}

func dashboardFields(birthday *entity.Birthday) []discordgo.MessageComponent {
	if birthday == nil {
		return []discordgo.MessageComponent{discordgo.TextDisplay{Content: fmt.Sprintf(
			"**%s**\n%s\n\n**%s**\n%s\n\n**%s**\n%s\n\n**%s**\n%s\n\n**%s**\n%s",
			ptbr.BirthdayUserIDLabel, emptyDashboardValue,
			ptbr.BirthdayNameLabel, emptyDashboardValue,
			ptbr.BirthdayDateLabel, emptyDashboardValue,
			ptbr.BirthdayTimeZoneLabel, emptyDashboardValue,
			ptbr.BirthdayMessageLabel, emptyDashboardValue,
		)}}
	}
	message := birthday.Message
	if message == "" {
		message = ptbr.BirthdayDefaultMessageValue
	}
	return []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: fmt.Sprintf("**%s**\n`%d`", ptbr.BirthdayUserIDLabel, birthday.UserID)},
		editableSection(ptbr.BirthdayNameLabel, escapeDisplayValue(birthday.Name), editFieldName, birthday.UserID),
		editableSection(ptbr.BirthdayDateLabel, birthday.Birthday.Format(birthdayDateFormat), editFieldBirthday, birthday.UserID),
		editableSection(ptbr.BirthdayTimeZoneLabel, "`"+birthday.TimeZone+"`", editFieldTimeZone, birthday.UserID),
		editableSection(ptbr.BirthdayMessageLabel, escapeDisplayValue(message), editFieldMessage, birthday.UserID),
	}
}

func editableSection(label, value, field string, userID uint64) discordgo.Section {
	return discordgo.Section{
		Components: []discordgo.MessageComponent{discordgo.TextDisplay{Content: fmt.Sprintf("**%s**\n%s", label, value)}},
		Accessory: discordgo.Button{
			Style:    discordgo.SecondaryButton,
			Emoji:    &discordgo.ComponentEmoji{Name: "✏️"},
			CustomID: fmt.Sprintf("%s:%s:%d", editFieldRoute, field, userID),
		},
	}
}

func dashboardSelect(selectedUserID uint64) discordgo.ActionsRow {
	menu := discordgo.SelectMenu{
		MenuType:    discordgo.UserSelectMenu,
		CustomID:    editSelectRoute,
		Placeholder: ptbr.BirthdayEditSelectPlaceholder,
		MinValues:   new(1),
		MaxValues:   1,
	}
	if selectedUserID != 0 {
		menu.DefaultValues = []discordgo.SelectMenuDefaultValue{{
			ID: strconv.FormatUint(selectedUserID, 10), Type: discordgo.SelectMenuDefaultValueUser,
		}}
	}
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{menu}}
}

func componentResponse(
	responseType discordgo.InteractionResponseType,
	ephemeral bool,
	components ...discordgo.MessageComponent,
) *discordgo.InteractionResponse {
	flags := discordgo.MessageFlagsIsComponentsV2
	if ephemeral {
		flags |= discordgo.MessageFlagsEphemeral
	}
	return &discordgo.InteractionResponse{
		Type: responseType,
		Data: &discordgo.InteractionResponseData{
			Flags: flags, AllowedMentions: &discordgo.MessageAllowedMentions{}, Components: components,
		},
	}
}

func escapeDisplayValue(value string) string {
	return strings.NewReplacer(
		"\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "~", "\\~",
		"|", "\\|", ">", "\\>", "#", "\\#", "[", "\\[", "]", "\\]",
	).Replace(value)
}
