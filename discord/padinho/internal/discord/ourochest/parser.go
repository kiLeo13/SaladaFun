package ourochest

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	appourochest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ourochest"
)

var commandPattern = regexp.MustCompile(`(?i)^\$(oc|ourochest|oh|ouroharvest)(?:\s+([1-9]|10))?$`)
var ocComponentPattern = regexp.MustCompile(`(?:^|[^a-z0-9])oc(?:[^a-z0-9]|$)`)
var ohComponentPattern = regexp.MustCompile(`(?:^|[^a-z0-9])oh(?:[^a-z0-9]|$)`)

type gameKind uint8

const (
	gameUnknown gameKind = iota
	gameOC
	gameOH
)

type pendingCommand struct {
	kind      gameKind
	remaining int
	recorded  time.Time
}

type boardSnapshot struct {
	board       appourochest.Board
	unavailable [appourochest.CellCount]bool
	fingerprint string
	revealed    int
	terminal    bool
}

// parseCommand recognizes only complete supported Mudae game commands.
func parseCommand(content string) (gameKind, int, bool) {
	match := commandPattern.FindStringSubmatch(strings.TrimSpace(content))
	if match == nil {
		return gameUnknown, 0, false
	}
	kind := gameOH
	if strings.EqualFold(match[1], "oc") || strings.EqualFold(match[1], "ourochest") {
		kind = gameOC
	}
	uses := 1
	if match[2] != "" {
		parsed, err := strconv.Atoi(match[2])
		if err != nil {
			return gameUnknown, 0, false
		}
		uses = parsed
	}
	return kind, uses, true
}

// classifyBoardMessage extracts an explicit game signature from Mudae's payload.
func classifyBoardMessage(message *discordgo.Message) gameKind {
	text := strings.ToLower(messageText(message))
	oc := containsAny(text, "$oc", "ourochest", "red sphere", "esfera vermelha", "esfera roja", "sphère rouge")
	oh := containsAny(text, "$oh", "ouroharvest", "sphere harvest", "colheita de esferas")

	componentIDs := strings.ToLower(componentCustomIDs(message.Components))
	oc = oc || ocComponentPattern.MatchString(componentIDs)
	oh = oh || ohComponentPattern.MatchString(componentIDs)
	if oc == oh {
		return gameUnknown
	}
	if oc {
		return gameOC
	}
	return gameOH
}

// parseBoard flattens exactly five rows of five buttons and maps configured emojis.
func parseBoard(message *discordgo.Message, emojiColors map[string]appourochest.Color) (boardSnapshot, bool) {
	buttons, ok := flattenButtons(message.Components)
	if !ok {
		return boardSnapshot{}, false
	}

	var snapshot boardSnapshot
	var fingerprint strings.Builder
	allDisabled := true
	for position, button := range buttons {
		emojiID := ""
		if button.Emoji != nil {
			emojiID = button.Emoji.ID
		}
		if color, exists := emojiColors[emojiID]; exists {
			snapshot.board[position] = color
			snapshot.revealed++
		}
		snapshot.unavailable[position] = button.Disabled
		allDisabled = allDisabled && button.Disabled
		fingerprint.WriteString(emojiID)
		fingerprint.WriteByte(':')
		if button.Disabled {
			fingerprint.WriteByte('1')
		} else {
			fingerprint.WriteByte('0')
		}
		fingerprint.WriteByte('|')
	}
	snapshot.fingerprint = fingerprint.String()
	snapshot.terminal = snapshot.revealed >= appourochest.MaxClicks || allDisabled
	return snapshot, true
}

// flattenButtons validates the legacy Discord layout used by Mudae's 5x5 games.
func flattenButtons(components []discordgo.MessageComponent) ([]discordgo.Button, bool) {
	if len(components) != appourochest.BoardWidth {
		return nil, false
	}
	buttons := make([]discordgo.Button, 0, appourochest.CellCount)
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
		if len(children) != appourochest.BoardWidth {
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
	return buttons, len(buttons) == appourochest.CellCount
}

// messageText collects the stable textual surfaces that can identify a game.
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

// componentCustomIDs collects button identifiers for explicit mode signatures.
func componentCustomIDs(components []discordgo.MessageComponent) string {
	buttons, ok := flattenButtons(components)
	if !ok {
		return ""
	}
	var result strings.Builder
	for _, button := range buttons {
		result.WriteByte(' ')
		result.WriteString(button.CustomID)
	}
	return result.String()
}

// containsAny reports whether text contains at least one marker.
func containsAny(text string, markers ...string) bool {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
