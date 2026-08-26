package ourochest

import (
	"fmt"
	"testing"

	"github.com/bwmarrin/discordgo"
	appourochest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ourochest"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		content string
		kind    gameKind
		uses    int
		ok      bool
	}{
		{"$oc", gameOC, 1, true}, {" $ourochest 10 ", gameOC, 10, true},
		{"$OH 3", gameOH, 3, true}, {"$ouroharvest", gameOH, 1, true},
		{"please use $oc", gameUnknown, 0, false}, {"$oc 11", gameUnknown, 0, false},
	}
	for _, test := range tests {
		kind, uses, ok := parseCommand(test.content)
		if kind != test.kind || uses != test.uses || ok != test.ok {
			t.Fatalf("parseCommand(%q) = %d, %d, %t", test.content, kind, uses, ok)
		}
	}
}

func TestClassifyBoardMessage(t *testing.T) {
	tests := []struct {
		message *discordgo.Message
		want    gameKind
	}{
		{&discordgo.Message{Content: "Find the red sphere"}, gameOC},
		{&discordgo.Message{Embeds: []*discordgo.MessageEmbed{{Title: "$oh sphere harvest"}}}, gameOH},
		{testBoardMessage("board", "", "oc:cell"), gameOC},
		{testBoardMessage("board", "", "oh:cell"), gameOH},
		{testBoardMessage("board", "", "grid-cell"), gameUnknown},
	}
	for index, test := range tests {
		if got := classifyBoardMessage(test.message); got != test.want {
			t.Fatalf("case %d classification = %d, want %d", index, got, test.want)
		}
	}
}

func TestParseBoardMapsCustomEmojiIDs(t *testing.T) {
	message := testBoardMessage("board", "", "grid-cell")
	buttons, ok := flattenButtons(message.Components)
	if !ok {
		t.Fatal("flattenButtons() = false")
	}
	buttons[0].Emoji = &discordgo.ComponentEmoji{ID: "101"}
	buttons[0].Disabled = true
	message.Components = buttonRows(buttons)
	snapshot, ok := parseBoard(message, map[string]appourochest.Color{"101": appourochest.Blue})
	if !ok || snapshot.board[0] != appourochest.Blue || !snapshot.unavailable[0] || snapshot.revealed != 1 || snapshot.terminal {
		t.Fatalf("snapshot = %#v, %t", snapshot, ok)
	}
}

func TestParseBoardDetectsTerminalStates(t *testing.T) {
	message := testBoardMessage("board", "", "grid-cell")
	buttons, _ := flattenButtons(message.Components)
	colors := []string{"101", "102", "103", "104", "105"}
	colorMap := make(map[string]appourochest.Color)
	for index, id := range colors {
		buttons[index].Emoji = &discordgo.ComponentEmoji{ID: id}
		colorMap[id] = appourochest.Color(index + 1)
	}
	message.Components = buttonRows(buttons)
	snapshot, ok := parseBoard(message, colorMap)
	if !ok || !snapshot.terminal {
		t.Fatalf("snapshot = %#v, %t", snapshot, ok)
	}

	message = testBoardMessage("disabled", "", "grid-cell")
	buttons, _ = flattenButtons(message.Components)
	for index := range buttons {
		buttons[index].Disabled = true
	}
	message.Components = buttonRows(buttons)
	snapshot, ok = parseBoard(message, colorMap)
	if !ok || !snapshot.terminal {
		t.Fatalf("disabled snapshot = %#v, %t", snapshot, ok)
	}
}

func TestFlattenButtonsRejectsWrongShapes(t *testing.T) {
	if _, ok := flattenButtons(nil); ok {
		t.Fatal("flattenButtons(nil) = true")
	}
	message := testBoardMessage("board", "", "grid-cell")
	message.Components = message.Components[:4]
	if _, ok := flattenButtons(message.Components); ok {
		t.Fatal("flattenButtons(four rows) = true")
	}
	message.Components = []discordgo.MessageComponent{discordgo.Button{CustomID: "button"}}
	if _, ok := flattenButtons(message.Components); ok {
		t.Fatal("flattenButtons(button row) = true")
	}
}

func testBoardMessage(id, content, customIDPrefix string) *discordgo.Message {
	buttons := make([]discordgo.Button, appourochest.CellCount)
	for position := range buttons {
		buttons[position] = discordgo.Button{
			CustomID: fmt.Sprintf("%s:%d", customIDPrefix, position),
			Emoji:    &discordgo.ComponentEmoji{ID: "999"},
		}
	}
	return &discordgo.Message{
		ID: id, ChannelID: "300", GuildID: "400", Content: content,
		Author: &discordgo.User{ID: "100", Bot: true}, Components: buttonRows(buttons),
	}
}

func buttonRows(buttons []discordgo.Button) []discordgo.MessageComponent {
	rows := make([]discordgo.MessageComponent, 0, appourochest.BoardWidth)
	for row := 0; row < appourochest.BoardWidth; row++ {
		children := make([]discordgo.MessageComponent, 0, appourochest.BoardWidth)
		for column := 0; column < appourochest.BoardWidth; column++ {
			children = append(children, buttons[row*appourochest.BoardWidth+column])
		}
		rows = append(rows, discordgo.ActionsRow{Components: children})
	}
	return rows
}
