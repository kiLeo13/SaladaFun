package accounttree

import (
	"errors"
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

func TestTreeBuildsCompleteRootedHierarchy(t *testing.T) {
	rootID, childID, grandchildID := uint64(1), uint64(2), uint64(3)
	service := NewService(fakeRepository{links: []entity.DiscordAccountLink{
		{UserID: rootID}, {UserID: childID, ParentID: &rootID}, {UserID: grandchildID, ParentID: &childID},
	}})
	tree, err := service.Tree(grandchildID)
	if err != nil || tree.Count != 3 || tree.Root.UserID != rootID || len(tree.Root.Children) != 1 ||
		tree.Root.Children[0].UserID != childID || len(tree.Root.Children[0].Children) != 1 ||
		tree.Root.Children[0].Children[0].UserID != grandchildID {
		t.Fatalf("Tree() = %#v, %v", tree, err)
	}
}

func TestTreeReturnsUnlinkedUserAndPropagatesRepositoryFailure(t *testing.T) {
	service := NewService(fakeRepository{})
	tree, err := service.Tree(10)
	if err != nil || tree.Count != 1 || tree.Root.UserID != 10 {
		t.Fatalf("unlinked Tree() = %#v, %v", tree, err)
	}
	want := errors.New("database unavailable")
	if _, err := NewService(fakeRepository{err: want}).Tree(10); !errors.Is(err, want) {
		t.Fatalf("Tree() error = %v", err)
	}
}

func TestTreeRejectsMalformedRows(t *testing.T) {
	rootID := uint64(1)
	for _, links := range [][]entity.DiscordAccountLink{
		{{UserID: rootID}, {UserID: rootID}},
		{{UserID: rootID}, {UserID: 2, ParentID: uint64Pointer(3)}},
		{{UserID: rootID}, {UserID: 2}},
	} {
		if _, err := NewService(fakeRepository{links: links}).Tree(rootID); !errors.Is(err, ErrInvalidTree) {
			t.Fatalf("Tree(%#v) error = %v", links, err)
		}
	}
}

type fakeRepository struct {
	links []entity.DiscordAccountLink
	err   error
}

// Tree returns the configured hierarchy rows.
func (r fakeRepository) Tree(uint64) ([]entity.DiscordAccountLink, error) {
	return r.links, r.err
}

func uint64Pointer(value uint64) *uint64 { return &value }
