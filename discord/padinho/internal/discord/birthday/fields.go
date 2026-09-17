package birthday

import (
	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const (
	nameInputID          = "name"
	userInputID          = "user"
	birthdayInputID      = "birthday"
	timeZoneInputID      = "time_zone"
	messageInputID       = "message"
	maximumNameLength    = 100
	birthdayDateFormat   = "02/01/2006"
	birthdayDateLength   = len(birthdayDateFormat)
	maximumMessageLength = 1800
	brasiliaTimeZone     = "America/Sao_Paulo"
	amazonasTimeZone     = "America/Manaus"
	utcTimeZone          = "UTC"

	editFieldName     = nameInputID
	editFieldBirthday = birthdayInputID
	editFieldTimeZone = timeZoneInputID
	editFieldMessage  = messageInputID
)

type birthdayFieldDefinition struct {
	id            string
	label         string
	placeholder   string
	style         discordgo.TextInputStyle
	required      bool
	maximumLength int
	selectMenu    bool
}

// birthdayFieldDefinitions returns the ordered modal definitions shared by
// birthday creation and editing.
func birthdayFieldDefinitions() []birthdayFieldDefinition {
	return []birthdayFieldDefinition{
		{
			id: nameInputID, label: ptbr.BirthdayNameLabel, placeholder: ptbr.BirthdayNamePlaceholder,
			style: discordgo.TextInputShort, required: true, maximumLength: maximumNameLength,
		},
		{
			id: birthdayInputID, label: ptbr.BirthdayDateLabel, placeholder: ptbr.BirthdayDatePlaceholder,
			style: discordgo.TextInputShort, required: true, maximumLength: birthdayDateLength,
		},
		{
			id: timeZoneInputID, label: ptbr.BirthdayTimeZoneLabel, placeholder: ptbr.BirthdayTimeZonePlaceholder,
			required: true, selectMenu: true,
		},
		{
			id: messageInputID, label: ptbr.BirthdayMessageLabel, placeholder: ptbr.BirthdayMessagePlaceholder,
			style: discordgo.TextInputParagraph, required: false, maximumLength: maximumMessageLength,
		},
	}
}

// birthdayField finds one shared modal definition by its stable field ID.
func birthdayField(fieldID string) (birthdayFieldDefinition, bool) {
	for _, definition := range birthdayFieldDefinitions() {
		if definition.id == fieldID {
			return definition, true
		}
	}
	return birthdayFieldDefinition{}, false
}

// modalFieldLabel builds the Discord Label and control for one birthday field.
func modalFieldLabel(definition birthdayFieldDefinition, customID, value string) discordgo.Label {
	if definition.selectMenu {
		return timeZoneLabel(customID, value)
	}
	required := definition.required
	return discordgo.Label{
		Label: definition.label,
		Component: discordgo.TextInput{
			CustomID: customID, Value: value, Placeholder: definition.placeholder,
			Style: definition.style, Required: &required, MaxLength: definition.maximumLength,
		},
	}
}

// timeZoneLabel builds the shared timezone selector and marks the current
// value as selected. New registrations default to Brasilia.
func timeZoneLabel(customID, selected string) discordgo.Label {
	if selected == "" {
		selected = brasiliaTimeZone
	}
	required := true
	return discordgo.Label{
		Label: ptbr.BirthdayTimeZoneLabel,
		Component: discordgo.SelectMenu{
			MenuType: discordgo.StringSelectMenu, CustomID: customID,
			Placeholder: ptbr.BirthdayTimeZonePlaceholder,
			MinValues:   new(1), MaxValues: 1, Required: &required,
			Options: []discordgo.SelectMenuOption{
				{Label: ptbr.BirthdayTimeZoneBrasilia, Value: brasiliaTimeZone, Default: selected == brasiliaTimeZone},
				{Label: ptbr.BirthdayTimeZoneAmazonas, Value: amazonasTimeZone, Default: selected == amazonasTimeZone},
				{Label: ptbr.BirthdayTimeZoneUTC, Value: utcTimeZone, Default: selected == utcTimeZone},
			},
		},
	}
}
