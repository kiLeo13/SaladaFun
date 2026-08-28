package ouroharvest

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	appouroharvest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ouroharvest"
)

var darkPurplePattern = regexp.MustCompile(`(?is)\bspD\b.{0,80}?(?:turns\s+into|se\s+transforma\s+em|vira).{0,80}?\bspP\b`)

type boardSnapshot struct {
	colors      [appouroharvest.CellCount]appouroharvest.Color
	disabled    [appouroharvest.CellCount]bool
	state       appouroharvest.State
	fingerprint string
	terminal    bool
}

// parseBoard validates a complete Mudae grid and derives its current policy state.
func parseBoard(message *discordgo.Message, colors map[string]appouroharvest.Color) (boardSnapshot, bool) {
	buttons, ok := flattenButtons(message.Components)
	if !ok {
		return boardSnapshot{}, false
	}
	var snapshot boardSnapshot
	var fingerprint strings.Builder
	paidClicks := 0
	disabledDark := 0
	allDisabled := true
	for position, button := range buttons {
		if button.Emoji == nil {
			return boardSnapshot{}, false
		}
		color, exists := colors[button.Emoji.ID]
		if !exists {
			return boardSnapshot{}, false
		}
		snapshot.colors[position] = color
		snapshot.disabled[position] = button.Disabled
		allDisabled = allDisabled && button.Disabled
		fmt.Fprintf(&fingerprint, "%s:%t|", button.Emoji.ID, button.Disabled)
		if button.Disabled {
			if color != appouroharvest.Purple {
				paidClicks++
			}
			if color == appouroharvest.Dark {
				disabledDark++
			}
			if color == appouroharvest.Covered {
				snapshot.state.ChestFound = true
			}
			continue
		}
		addEnabledColor(&snapshot.state, color)
	}
	refunds := len(darkPurplePattern.FindAllString(messageText(message), -1))
	if refunds > disabledDark {
		refunds = disabledDark
	}
	paidClicks -= refunds
	if paidClicks < 0 || paidClicks > appouroharvest.PaidClickLimit {
		return boardSnapshot{}, false
	}
	snapshot.state.ClicksLeft = uint8(appouroharvest.PaidClickLimit - paidClicks)
	snapshot.fingerprint = fmt.Sprintf("%s%d:%t", fingerprint.String(), refunds, snapshot.state.ChestFound)
	snapshot.terminal = allDisabled || snapshot.state.ClicksLeft == 0
	return snapshot, true
}

// addEnabledColor includes a selectable button in the compressed state.
func addEnabledColor(state *appouroharvest.State, color appouroharvest.Color) {
	switch color {
	case appouroharvest.Covered:
		state.Covered++
	case appouroharvest.Blue:
		state.Blue++
	case appouroharvest.Teal:
		state.Teal++
	case appouroharvest.Dark:
		state.Dark++
	case appouroharvest.Green:
		state.Green++
	case appouroharvest.Yellow:
		state.Yellow++
	case appouroharvest.Light:
		state.Light++
	case appouroharvest.Orange:
		state.Orange++
	case appouroharvest.Red:
		state.Red++
	case appouroharvest.White:
		state.White++
	}
}

// positionFor returns the first enabled physical button matching an action.
func (s boardSnapshot) positionFor(action appouroharvest.Action) (int, bool) {
	for position, color := range s.colors {
		if !s.disabled[position] && color == appouroharvest.Color(action) {
			return position, true
		}
	}
	return 0, false
}

// purplePosition returns the first free purple button awaiting collection.
func (s boardSnapshot) purplePosition() (int, bool) {
	for position, color := range s.colors {
		if !s.disabled[position] && color == appouroharvest.Purple {
			return position, true
		}
	}
	return 0, false
}

// flattenButtons validates Mudae's legacy five-row component layout.
func flattenButtons(components []discordgo.MessageComponent) ([]discordgo.Button, bool) {
	if len(components) != appouroharvest.BoardWidth {
		return nil, false
	}
	buttons := make([]discordgo.Button, 0, appouroharvest.CellCount)
	for _, component := range components {
		var children []discordgo.MessageComponent
		switch row := component.(type) {
		case discordgo.ActionsRow:
			children = row.Components
		case *discordgo.ActionsRow:
			children = row.Components
		default:
			return nil, false
		}
		if len(children) != appouroharvest.BoardWidth {
			return nil, false
		}
		for _, child := range children {
			switch button := child.(type) {
			case discordgo.Button:
				buttons = append(buttons, button)
			case *discordgo.Button:
				buttons = append(buttons, *button)
			default:
				return nil, false
			}
		}
	}
	return buttons, len(buttons) == appouroharvest.CellCount
}

// messageText combines every textual surface used for classification and refunds.
func messageText(message *discordgo.Message) string {
	var result strings.Builder
	result.WriteString(message.Content)
	for _, embed := range message.Embeds {
		result.WriteByte(' ')
		result.WriteString(embed.Title)
		result.WriteByte(' ')
		result.WriteString(embed.Description)
		for _, field := range embed.Fields {
			result.WriteByte(' ')
			result.WriteString(field.Name)
			result.WriteByte(' ')
			result.WriteString(field.Value)
		}
	}
	return result.String()
}

// isOuroharvestBoard recognizes stable Portuguese and English game instructions.
func isOuroharvestBoard(message *discordgo.Message) bool {
	text := strings.ToLower(strings.ReplaceAll(messageText(message), "**", ""))
	portuguese := strings.Contains(text, "você pode clicar 5 vezes") &&
		strings.Contains(text, "por 2 minutos") && strings.Contains(text, "esferas azuis revelam 3") &&
		strings.Contains(text, "esferas cianas revelam 1")
	english := strings.Contains(text, "you can click 5 times") &&
		strings.Contains(text, "for 2 minutes") && strings.Contains(text, "blue spheres reveal 3") &&
		strings.Contains(text, "teal spheres reveal 1")
	return portuguese || english
}
