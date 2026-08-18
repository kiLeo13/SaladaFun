package command

// OptionType identifies a Discord application-command option kind without
// exposing DiscordGo types to the command framework.
type OptionType uint8

const (
	OptionTypeString OptionType = iota + 1
	OptionTypeInteger
	OptionTypeBoolean
	OptionTypeUser
	OptionTypeChannel
)

// ChannelType identifies a supported channel type filter for channel options.
type ChannelType uint8

const (
	ChannelTypeVoice ChannelType = iota + 1
)

// OptionChoice is one selectable value for a string command option.
type OptionChoice struct {
	// Name is the human-readable label shown by Discord.
	Name string
	// Value is the string sent back in the interaction when selected.
	Value string
}

// OptionDefinition is the immutable option metadata emitted by a frozen
// registry.
type OptionDefinition struct {
	Type         OptionType
	Name         string
	Description  string
	Required     bool
	Autocomplete bool
	Choices      []OptionChoice
	ChannelTypes []ChannelType
}

// Option is a typed command-option descriptor.
type Option interface {
	snapshot() OptionDefinition
}

type optionDescriptor struct {
	definition OptionDefinition
}

func (o *optionDescriptor) markRequired() {
	o.definition.Required = true
}

func (o *optionDescriptor) snapshot() OptionDefinition {
	result := o.definition
	result.Choices = append([]OptionChoice(nil), o.definition.Choices...)
	result.ChannelTypes = append([]ChannelType(nil), o.definition.ChannelTypes...)
	return result
}

// StringCommandOption describes a string option.
type StringCommandOption struct{ optionDescriptor }

// StringOption creates a string option descriptor.
func StringOption(name, description string) *StringCommandOption {
	return &StringCommandOption{optionDescriptor{definition: OptionDefinition{
		Type: OptionTypeString, Name: name, Description: description,
	}}}
}

func (o *StringCommandOption) snapshot() OptionDefinition { return o.optionDescriptor.snapshot() }

// Required marks the option as required.
func (o *StringCommandOption) Required() *StringCommandOption {
	o.markRequired()
	return o
}

// Choices adds the fixed values displayed by Discord for this string option.
func (o *StringCommandOption) Choices(choices ...OptionChoice) *StringCommandOption {
	o.definition.Choices = append([]OptionChoice(nil), choices...)
	return o
}

// Autocomplete enables Discord autocomplete for the option.
func (o *StringCommandOption) Autocomplete() *StringCommandOption {
	o.definition.Autocomplete = true
	return o
}

// IntegerCommandOption describes an integer option.
type IntegerCommandOption struct{ optionDescriptor }

// IntegerOption creates an integer option descriptor.
func IntegerOption(name, description string) *IntegerCommandOption {
	return &IntegerCommandOption{optionDescriptor{definition: OptionDefinition{
		Type: OptionTypeInteger, Name: name, Description: description,
	}}}
}

func (o *IntegerCommandOption) snapshot() OptionDefinition { return o.optionDescriptor.snapshot() }

func (o *IntegerCommandOption) Required() *IntegerCommandOption {
	o.markRequired()
	return o
}

func (o *IntegerCommandOption) Autocomplete() *IntegerCommandOption {
	o.definition.Autocomplete = true
	return o
}

// BooleanCommandOption describes a Boolean option.
type BooleanCommandOption struct{ optionDescriptor }

// BooleanOption creates a Boolean option descriptor.
func BooleanOption(name, description string) *BooleanCommandOption {
	return &BooleanCommandOption{optionDescriptor{definition: OptionDefinition{
		Type: OptionTypeBoolean, Name: name, Description: description,
	}}}
}

func (o *BooleanCommandOption) snapshot() OptionDefinition { return o.optionDescriptor.snapshot() }

func (o *BooleanCommandOption) Required() *BooleanCommandOption {
	o.markRequired()
	return o
}

// UserCommandOption describes a Discord user option.
type UserCommandOption struct{ optionDescriptor }

// UserOption creates a user option descriptor.
func UserOption(name, description string) *UserCommandOption {
	return &UserCommandOption{optionDescriptor{definition: OptionDefinition{
		Type: OptionTypeUser, Name: name, Description: description,
	}}}
}

func (o *UserCommandOption) snapshot() OptionDefinition { return o.optionDescriptor.snapshot() }

func (o *UserCommandOption) Required() *UserCommandOption {
	o.markRequired()
	return o
}

// ChannelCommandOption describes a Discord channel option.
type ChannelCommandOption struct{ optionDescriptor }

// ChannelOption creates a channel option descriptor.
func ChannelOption(name, description string) *ChannelCommandOption {
	return &ChannelCommandOption{optionDescriptor{definition: OptionDefinition{
		Type: OptionTypeChannel, Name: name, Description: description,
	}}}
}

func (o *ChannelCommandOption) snapshot() OptionDefinition { return o.optionDescriptor.snapshot() }

func (o *ChannelCommandOption) Required() *ChannelCommandOption {
	o.markRequired()
	return o
}

// VoiceOnly restricts the channel option to guild voice channels.
func (o *ChannelCommandOption) VoiceOnly() *ChannelCommandOption {
	o.definition.ChannelTypes = []ChannelType{ChannelTypeVoice}
	return o
}
