package accounttree

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	appaccounttree "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/accounttree"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const maxMessageTextRunes = 4000

// render builds the compact, non-pinging Components V2 hierarchy message.
func render(tree *appaccounttree.Tree, names map[uint64]string) []discordgo.MessageComponent {
	title := fmt.Sprintf(ptbr.AccountTreeTitle, tree.Root.UserID)
	body := treeContent(tree, names, maxMessageTextRunes-utf8.RuneCountInString(title))
	return []discordgo.MessageComponent{discordgo.Container{Components: []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: title},
		discordgo.TextDisplay{Content: body},
	}}}
}

// treeContent formats complete tree lines until the Components V2 text budget is reached.
func treeContent(tree *appaccounttree.Tree, names map[uint64]string, budget int) string {
	footer := accountCountFooter(tree.Count)
	suffix := "\n```\n" + footer
	available := max(budget-utf8.RuneCountInString("```\n")-utf8.RuneCountInString(suffix), 0)
	lines := []string{safeName(names[tree.Root.UserID])}
	appendChildren(tree.Root, 1, names, &lines)
	visible := lines
	if lineRunes(lines) > available {
		visible = visibleLines(lines, available)
	}
	return "```\n" + strings.Join(visible, "\n") + suffix
}

// appendChildren adds every child line in display-name order.
func appendChildren(node *appaccounttree.Node, depth int, names map[uint64]string, lines *[]string) {
	children := append([]*appaccounttree.Node(nil), node.Children...)
	sort.SliceStable(children, func(left, right int) bool {
		leftName := strings.ToLower(safeName(names[children[left].UserID]))
		rightName := strings.ToLower(safeName(names[children[right].UserID]))
		if leftName == rightName {
			return children[left].UserID < children[right].UserID
		}
		return leftName < rightName
	})
	for _, child := range children {
		line := strings.Repeat("    ", depth-1) + "|__ " + safeName(names[child.UserID])
		*lines = append(*lines, line)
		appendChildren(child, depth+1, names, lines)
	}
}

// visibleLines keeps complete lines and reserves room for a truncation marker.
func visibleLines(lines []string, budget int) []string {
	visible := make([]string, 0, len(lines))
	for index, line := range lines {
		candidate := append(append([]string(nil), visible...), line)
		if index < len(lines)-1 {
			candidate = append(candidate, "...")
		}
		if lineRunes(candidate) > budget {
			break
		}
		visible = append(visible, line)
	}
	if len(visible) == 0 {
		return []string{"..."}
	}
	return append(visible, "...")
}

// lineRunes returns a multi-line display's rune length.
func lineRunes(lines []string) int {
	length := 0
	for index, line := range lines {
		length += utf8.RuneCountInString(line)
		if index > 0 {
			length++
		}
	}
	return length
}

// safeName keeps Discord member data inside the intended Markdown code block.
func safeName(value string) string {
	value = strings.NewReplacer("`", "'", "\r", " ", "\n", " ").Replace(value)
	if value == "" {
		return "?"
	}
	return value
}

// accountCountFooter localizes the full number of accounts, including omitted lines.
func accountCountFooter(count int) string {
	if count == 1 {
		return ptbr.AccountTreeOneAccount
	}
	return fmt.Sprintf(ptbr.AccountTreeManyAccounts, count)
}
