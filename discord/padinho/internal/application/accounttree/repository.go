package accounttree

import "github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"

// Repository loads one complete Discord account hierarchy rooted at a requested
// account or any of its descendants.
type Repository interface {
	Tree(userID uint64) ([]entity.DiscordAccountLink, error)
}
