package command

// Definition is a frozen top-level application-command definition.
type Definition struct {
	Name        string
	Description string
	Options     []OptionDefinition
	Subcommands []SubcommandDefinition
	Groups      []SubcommandGroupDefinition
}

// SubcommandDefinition is a frozen direct or grouped subcommand definition.
type SubcommandDefinition struct {
	Name        string
	Description string
	Options     []OptionDefinition
}

// SubcommandGroupDefinition is a frozen Discord subcommand group.
type SubcommandGroupDefinition struct {
	Name        string
	Description string
	Subcommands []SubcommandDefinition
}

func cloneOptionDefinitions(options []OptionDefinition) []OptionDefinition {
	if options == nil {
		return nil
	}
	result := make([]OptionDefinition, len(options))
	for index, option := range options {
		result[index] = option
		result[index].Choices = append([]OptionChoice(nil), option.Choices...)
	}
	return result
}

func cloneDefinitions(definitions []*Definition) []*Definition {
	result := make([]*Definition, len(definitions))
	for index, definition := range definitions {
		result[index] = &Definition{
			Name:        definition.Name,
			Description: definition.Description,
			Options:     cloneOptionDefinitions(definition.Options),
			Subcommands: make([]SubcommandDefinition, len(definition.Subcommands)),
			Groups:      make([]SubcommandGroupDefinition, len(definition.Groups)),
		}
		for subIndex, subcommand := range definition.Subcommands {
			result[index].Subcommands[subIndex] = SubcommandDefinition{
				Name: subcommand.Name, Description: subcommand.Description,
				Options: cloneOptionDefinitions(subcommand.Options),
			}
		}
		for groupIndex, group := range definition.Groups {
			result[index].Groups[groupIndex] = SubcommandGroupDefinition{
				Name: group.Name, Description: group.Description,
				Subcommands: make([]SubcommandDefinition, len(group.Subcommands)),
			}
			for subIndex, subcommand := range group.Subcommands {
				result[index].Groups[groupIndex].Subcommands[subIndex] = SubcommandDefinition{
					Name: subcommand.Name, Description: subcommand.Description,
					Options: cloneOptionDefinitions(subcommand.Options),
				}
			}
		}
	}
	return result
}
