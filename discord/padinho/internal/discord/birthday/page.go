package birthday

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	appbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/birthday"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/theme"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const birthdayAccentColor = theme.AccentColor

func pageResponse(
	responseType discordgo.InteractionResponseType,
	month time.Month,
	birthdays []*entity.Birthday,
	next *appbirthday.UpcomingBirthday,
	fullDate bool,
) *discordgo.InteractionResponse {
	container := pageContainer(month, birthdays, next, fullDate)
	actions := pageActions(month, fullDate)

	return &discordgo.InteractionResponse{
		Type: responseType,
		Data: &discordgo.InteractionResponseData{
			Flags:           discordgo.MessageFlagsIsComponentsV2,
			AllowedMentions: &discordgo.MessageAllowedMentions{},
			Components:      []discordgo.MessageComponent{container, actions},
		},
	}
}

func pageContainer(month time.Month, birthdays []*entity.Birthday, next *appbirthday.UpcomingBirthday, fullDate bool) discordgo.Container {
	accent := birthdayAccentColor
	divider := true
	return discordgo.Container{
		AccentColor: &accent,
		Components: []discordgo.MessageComponent{
			pageHeading(month),
			discordgo.TextDisplay{Content: pageContent(birthdays, fullDate)},
			discordgo.Separator{Divider: &divider},
			discordgo.TextDisplay{Content: upcomingContent(next)},
		},
	}
}

func pageActions(month time.Month, fullDate bool) discordgo.ActionsRow {
	previous := pageButton(
		"⬅️",
		"previous",
		month,
		month == time.January,
		fullDate,
	)
	next := pageButton(
		"➡️",
		"next",
		month,
		month == time.December,
		fullDate,
	)

	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		previous,
		next,
		addButton(),
		editButton(),
	}}
}

func pageHeading(month time.Month) discordgo.Section {
	return discordgo.Section{
		Components: []discordgo.MessageComponent{discordgo.TextDisplay{Content: pageTitle(month)}},
		Accessory: discordgo.Button{
			Style:    discordgo.SecondaryButton,
			Emoji:    &discordgo.ComponentEmoji{Name: "🔍"},
			CustomID: inspectRoute,
		},
	}
}

func pageTitle(month time.Month) string {
	return fmt.Sprintf("## "+ptbr.BirthdayTitle, locale.Capitalize(ptbr.MonthNames[month]))
}

func pageContent(birthdays []*entity.Birthday, fullDate bool) string {
	var content strings.Builder
	if len(birthdays) == 0 {
		content.WriteString(ptbr.BirthdayEmptyMonth)
		return content.String()
	}
	for index, birthday := range birthdays {
		if index > 0 {
			content.WriteByte('\n')
		}
		if fullDate {
			fmt.Fprintf(
				&content,
				ptbr.BirthdayFullDateEntry,
				birthday.Birthday.Day(),
				birthday.Birthday.Month(),
				birthday.Birthday.Year(),
				birthday.UserID,
			)
			continue
		}
		fmt.Fprintf(&content, ptbr.BirthdayEntry, birthday.Birthday.Day(), birthday.Birthday.Month(), birthday.UserID)
	}
	return content.String()
}

func upcomingContent(next *appbirthday.UpcomingBirthday) string {
	if next == nil {
		return "-# " + ptbr.BirthdayNoUpcoming
	}
	return fmt.Sprintf(ptbr.BirthdayUpcoming, next.UserID, next.OccursAt.Unix())
}

func pageButton(
	emoji string,
	direction string,
	month time.Month,
	disabled bool,
	fullDate bool,
) discordgo.Button {
	return discordgo.Button{
		Style: discordgo.SecondaryButton, Disabled: disabled,
		Emoji:    &discordgo.ComponentEmoji{Name: emoji},
		CustomID: fmt.Sprintf("%s:%s:%d:%t", pageRoute, direction, month, fullDate),
	}
}

func addButton() discordgo.Button {
	return discordgo.Button{
		Style:    discordgo.SuccessButton,
		Label:    ptbr.BirthdayButtonAdd,
		CustomID: addBirthdayRoute,
	}
}

func editButton() discordgo.Button {
	return discordgo.Button{
		Style:    discordgo.PrimaryButton,
		Label:    ptbr.BirthdayButtonEdit,
		Emoji:    &discordgo.ComponentEmoji{Name: "✏️"},
		CustomID: editRoute,
	}
}

func parsePage(parameters []string) (string, time.Month, bool, error) {
	if len(parameters) != 2 && len(parameters) != 3 {
		return "", 0, false, fmt.Errorf("invalid birthday page parameters")
	}
	if parameters[0] != "previous" && parameters[0] != "next" {
		return "", 0, false, fmt.Errorf("invalid birthday page parameters")
	}
	month, err := strconv.Atoi(parameters[1])
	if err != nil || month < int(time.January) || month > int(time.December) {
		return "", 0, false, fmt.Errorf("invalid birthday page month")
	}
	fullDate := false
	if len(parameters) == 3 {
		fullDate, err = strconv.ParseBool(parameters[2])
		if err != nil {
			return "", 0, false, fmt.Errorf("invalid birthday page date format")
		}
	}
	return parameters[0], time.Month(month), fullDate, nil
}
