package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
)

// CompileDefinitions converts the framework-neutral registry snapshot at the
// Discord boundary.
func CompileDefinitions(definitions []*command.Definition) ([]*discordgo.ApplicationCommand, error) {
	result := make([]*discordgo.ApplicationCommand, 0, len(definitions))
	for _, definition := range definitions {
		compiled := &discordgo.ApplicationCommand{
			Type: discordgo.ChatApplicationCommand, Name: definition.Name,
			Description: definition.Description,
		}
		var err error
		compiled.Options, err = compileOptions(definition.Options)
		if err != nil {
			return nil, fmt.Errorf("compile /%s: %w", definition.Name, err)
		}
		for _, subcommand := range definition.Subcommands {
			options, optionErr := compileOptions(subcommand.Options)
			if optionErr != nil {
				return nil, fmt.Errorf("compile /%s %s: %w", definition.Name, subcommand.Name, optionErr)
			}
			compiled.Options = append(compiled.Options, &discordgo.ApplicationCommandOption{
				Type: discordgo.ApplicationCommandOptionSubCommand, Name: subcommand.Name,
				Description: subcommand.Description, Options: options,
			})
		}
		for _, group := range definition.Groups {
			compiledGroup := &discordgo.ApplicationCommandOption{
				Type: discordgo.ApplicationCommandOptionSubCommandGroup,
				Name: group.Name, Description: group.Description,
			}
			for _, subcommand := range group.Subcommands {
				options, optionErr := compileOptions(subcommand.Options)
				if optionErr != nil {
					return nil, fmt.Errorf("compile /%s %s %s: %w", definition.Name, group.Name, subcommand.Name, optionErr)
				}
				compiledGroup.Options = append(compiledGroup.Options, &discordgo.ApplicationCommandOption{
					Type: discordgo.ApplicationCommandOptionSubCommand, Name: subcommand.Name,
					Description: subcommand.Description, Options: options,
				})
			}
			compiled.Options = append(compiled.Options, compiledGroup)
		}
		result = append(result, compiled)
	}
	return result, nil
}

func compileOptions(options []command.OptionDefinition) ([]*discordgo.ApplicationCommandOption, error) {
	result := make([]*discordgo.ApplicationCommandOption, 0, len(options))
	for _, option := range options {
		optionType, err := compileOptionType(option.Type)
		if err != nil {
			return nil, err
		}
		result = append(result, &discordgo.ApplicationCommandOption{
			Type: optionType, Name: option.Name, Description: option.Description,
			Required: option.Required, Autocomplete: option.Autocomplete,
		})
	}
	return result, nil
}

func compileOptionType(optionType command.OptionType) (discordgo.ApplicationCommandOptionType, error) {
	switch optionType {
	case command.OptionTypeString:
		return discordgo.ApplicationCommandOptionString, nil
	case command.OptionTypeInteger:
		return discordgo.ApplicationCommandOptionInteger, nil
	case command.OptionTypeBoolean:
		return discordgo.ApplicationCommandOptionBoolean, nil
	case command.OptionTypeUser:
		return discordgo.ApplicationCommandOptionUser, nil
	case command.OptionTypeChannel:
		return discordgo.ApplicationCommandOptionChannel, nil
	default:
		return 0, fmt.Errorf("unsupported command option type %d", optionType)
	}
}
