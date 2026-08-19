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

func inspectionResponse(birthday *entity.Birthday, guild *discordgo.Guild) *discordgo.InteractionResponse {
	accent := birthdayAccentColor
	message := birthday.Message
	messageDisplay := escapeDisplayValue(message)
	if message == "" {
		message = ptbr.BirthdayDefaultMessageValue
		messageDisplay = message
	}

	embed := discordgo.MessageEmbed{
		Color:       accent,
		Description: "## " + ptbr.BirthdayInspectTitle,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   ptbr.BirthdayNameLabel,
				Value:  escapeDisplayValue(birthday.Name),
				Inline: true,
			},
			{
				Name:   ptbr.BirthdayDateLabel,
				Value:  birthday.Birthday.Format(birthdayDateFormat),
				Inline: true,
			},
			{
				Name:   ptbr.BirthdayTimeZoneLabel,
				Value:  "`" + birthday.TimeZone + "`",
				Inline: true,
			},
			{
				Name:   ptbr.BirthdayMessageLabel,
				Value:  messageDisplay,
				Inline: false,
			},
		},
		Footer: inspectionFooter(guild),
	}
	return embedResponse(embed)
}

// inspectionFooter renders the source guild's name and icon when cached.
func inspectionFooter(guild *discordgo.Guild) *discordgo.MessageEmbedFooter {
	if guild == nil || guild.Name == "" {
		return &discordgo.MessageEmbedFooter{Text: ptbr.BirthdayGuildUnknown}
	}
	return &discordgo.MessageEmbedFooter{Text: guild.Name, IconURL: guild.IconURL("64")}
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
	components = append(components, discordgo.TextDisplay{Content: "### " + ptbr.BirthdayDashboardUserLabel})
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
	messageDisplay := escapeDisplayValue(message)
	if message == "" {
		message = ptbr.BirthdayDefaultMessageValue
		messageDisplay = message
	}
	return []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: fmt.Sprintf("**%s**\n`%d`", ptbr.BirthdayUserIDLabel, birthday.UserID)},
		editableSection(ptbr.BirthdayNameLabel, escapeDisplayValue(birthday.Name), editFieldName, birthday.UserID),
		editableSection(ptbr.BirthdayDateLabel, birthday.Birthday.Format(birthdayDateFormat), editFieldBirthday, birthday.UserID),
		editableSection(ptbr.BirthdayTimeZoneLabel, "`"+birthday.TimeZone+"`", editFieldTimeZone, birthday.UserID),
		editableSection(ptbr.BirthdayMessageLabel, messageDisplay, editFieldMessage, birthday.UserID),
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

// embedResponse creates the private legacy embed used for birthday inspection.
func embedResponse(embed discordgo.MessageEmbed) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:           discordgo.MessageFlagsEphemeral,
			AllowedMentions: &discordgo.MessageAllowedMentions{},
			Embeds:          []*discordgo.MessageEmbed{&embed},
		},
	}
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
			Flags:           flags,
			AllowedMentions: &discordgo.MessageAllowedMentions{},
			Components:      components,
		},
	}
}

func escapeDisplayValue(value string) string {
	return strings.NewReplacer(
		"\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "~", "\\~",
		"|", "\\|", ">", "\\>", "#", "\\#", "[", "\\[", "]", "\\]",
	).Replace(value)
}
