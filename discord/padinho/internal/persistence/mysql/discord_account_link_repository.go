package mysql

import (
	"fmt"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"gorm.io/gorm"
)

const accountTreeQuery = `
WITH RECURSIVE ancestors (id, user_id, parent_id, depth) AS (
    SELECT id, user_id, parent_id, 0
    FROM discord_account_links
    WHERE user_id = ?

    UNION ALL

    SELECT parent.id, parent.user_id, parent.parent_id, ancestors.depth + 1
    FROM discord_account_links AS parent
    JOIN ancestors ON parent.user_id = ancestors.parent_id
), root_account AS (
    SELECT user_id
    FROM ancestors
    WHERE parent_id IS NULL
    LIMIT 1
), tree (id, user_id, parent_id, depth) AS (
    SELECT link.id, link.user_id, link.parent_id, 0
    FROM discord_account_links AS link
    JOIN root_account ON root_account.user_id = link.user_id

    UNION ALL

    SELECT child.id, child.user_id, child.parent_id, tree.depth + 1
    FROM discord_account_links AS child
    JOIN tree ON child.parent_id = tree.user_id
)
SELECT id, user_id, parent_id
FROM tree
ORDER BY depth, id`

// DiscordAccountLinkRepository loads sparse account hierarchies with MySQL.
type DiscordAccountLinkRepository struct {
	db *gorm.DB
}

// NewDiscordAccountLinkRepository binds account hierarchy reads to db.
func NewDiscordAccountLinkRepository(db *gorm.DB) *DiscordAccountLinkRepository {
	return &DiscordAccountLinkRepository{db: db}
}

// Tree returns the rooted hierarchy containing userID in one recursive query.
func (r *DiscordAccountLinkRepository) Tree(userID uint64) ([]entity.DiscordAccountLink, error) {
	var links []entity.DiscordAccountLink
	result := r.db.Raw(accountTreeQuery, userID).Scan(&links)
	if result.Error != nil {
		return nil, fmt.Errorf("query Discord account tree: %w", result.Error)
	}
	return links, nil
}
