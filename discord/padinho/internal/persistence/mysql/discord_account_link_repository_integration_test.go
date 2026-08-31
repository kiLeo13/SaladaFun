package mysql

import (
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

func TestDiscordAccountLinkRepositoryAgainstMySQL(t *testing.T) {
	setLiveEnvironment(t)
	db, err := database.Open()
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close(db) })
	transaction := db.Begin()
	if transaction.Error != nil {
		t.Fatalf("begin repository test: %v", transaction.Error)
	}
	t.Cleanup(func() { _ = transaction.Rollback().Error })
	rootID, childID, grandchildID := uint64(900000000000000110), uint64(900000000000000111), uint64(900000000000000112)
	root := entity.DiscordAccountLink{UserID: rootID}
	child := entity.DiscordAccountLink{UserID: childID, ParentID: &rootID}
	grandchild := entity.DiscordAccountLink{UserID: grandchildID, ParentID: &childID}
	for _, link := range []*entity.DiscordAccountLink{&root, &child, &grandchild} {
		if err := transaction.Create(link).Error; err != nil {
			t.Fatalf("create account link: %v", err)
		}
	}
	links, err := NewDiscordAccountLinkRepository(transaction).Tree(grandchildID)
	if err != nil || len(links) != 3 || links[0].UserID != rootID || links[1].UserID != childID || links[2].UserID != grandchildID {
		t.Fatalf("Tree() = %#v, %v", links, err)
	}
}
