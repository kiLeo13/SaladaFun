package command

import (
	"context"
	"testing"
)

// TestChoiceValidation exercises Discord constraints used by history ordering.
func TestChoiceValidation(t *testing.T) {
	for _, scenario := range []string{"too many", "non-string", "empty", "duplicate", "autocomplete"} {
		t.Run(scenario, func(t *testing.T) {
			r := NewRegistry()
			option := StringOption("sort", "Sort").Choices(OptionChoice{Name: "Date", Value: "date"})
			switch scenario {
			case "too many":
				choices := make([]OptionChoice, maximumOptionChoices+1)
				for i := range choices {
					choices[i] = OptionChoice{Name: "Date", Value: "date"}
				}
				option.Choices(choices...)
			case "non-string":
				option.definition.Type = OptionTypeInteger
			case "empty":
				option.Choices(OptionChoice{})
			case "duplicate":
				option.Choices(OptionChoice{Name: "Date", Value: "date"}, OptionChoice{Name: "Again", Value: "date"})
			case "autocomplete":
				option.Autocomplete()
			}
			r.Slash("test", "Test", func(context.Context, *CommandRequest) error { return nil }, option)
			if err := r.Freeze(); err == nil {
				t.Fatal("invalid choice declaration accepted")
			}
		})
	}
}

// TestFrozenChoiceMetadata keeps published definitions independent of callers.
func TestFrozenChoiceMetadata(t *testing.T) {
	r := NewRegistry()
	r.Slash("test", "Test", func(context.Context, *CommandRequest) error { return nil }, StringOption("sort", "Sort").Choices(OptionChoice{Name: "Date", Value: "date"}), ChannelOption("channel", "Channel").VoiceOnly())
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}
	definitions, _ := r.Definitions()
	definitions[0].Options[0].Choices[0].Value = "changed"
	again, _ := r.Definitions()
	if again[0].Options[0].Choices[0].Value != "date" || len(again[0].Options[1].ChannelTypes) != 1 {
		t.Fatal("mutable metadata")
	}
}
