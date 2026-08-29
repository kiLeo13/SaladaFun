// Package quote selects publishable quotes from the catalog.
package quote

import "github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"

// Repository is the persistence capability consumed by the quote service.
type Repository interface {
	Random() (*entity.Quote, error)
}
