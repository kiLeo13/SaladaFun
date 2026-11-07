package letterboxd

import (
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
)

// TestHistoryChoiceLabels preserves English command labels and stable option values.
func TestHistoryChoiceLabels(t *testing.T) {
	routes := discord.NewRoutes()
	Register(routes, nil)
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	definitions, err := routes.Commands().Definitions()
	if err != nil {
		t.Fatal(err)
	}
	commands, err := discord.CompileDefinitions(definitions)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]map[string]string{
		"sort":  {"date": "Date", "rating": "Rating"},
		"order": {"asc": "Ascending", "desc": "Descending"},
	}
	for _, sub := range commands[0].Options {
		if sub.Name != "history" {
			continue
		}
		for _, option := range sub.Options {
			labels, ok := expected[option.Name]
			if !ok {
				continue
			}
			if len(option.Choices) != len(labels) {
				t.Fatal("choice count", option.Name)
			}
			for _, choice := range option.Choices {
				value, ok := choice.Value.(string)
				if !ok || labels[value] != choice.Name {
					t.Fatalf("%s choice %#v", option.Name, choice)
				}
			}
			delete(expected, option.Name)
		}
	}
	if len(expected) != 0 {
		t.Fatal("missing history options", expected)
	}
}
