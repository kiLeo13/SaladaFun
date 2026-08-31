package accounttree

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	appaccounttree "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/accounttree"
)

func TestRenderSanitizesNamesAndUsesSingularFooter(t *testing.T) {
	tree := &appaccounttree.Tree{Root: &appaccounttree.Node{UserID: 1}, Count: 1}
	components := render(tree, map[uint64]string{1: "Mike```\nignored"})
	container := components[0].(discordgo.Container)
	body := container.Components[1].(discordgo.TextDisplay).Content
	if container.AccentColor != nil || len(container.Components) != 2 || !strings.Contains(body, "Mike''' ignored") || strings.Count(body, "```") != 2 || !strings.Contains(body, "-# 1 conta total.") {
		t.Fatalf("rendered components = %#v", components)
	}
}

func TestTreeContentTruncatesLongTreesWithEllipsis(t *testing.T) {
	root := &appaccounttree.Node{UserID: 1}
	for userID := uint64(2); userID < 100; userID++ {
		root.Children = append(root.Children, &appaccounttree.Node{UserID: userID})
	}
	tree := &appaccounttree.Tree{Root: root, Count: 99}
	names := map[uint64]string{1: "Mike"}
	for userID := uint64(2); userID < 100; userID++ {
		names[userID] = strings.Repeat("A", 40)
	}
	content := treeContent(tree, names, 100)
	if !strings.Contains(content, "...") || !strings.Contains(content, "-# Contas totais: 99.") {
		t.Fatalf("truncated content = %q", content)
	}
}
