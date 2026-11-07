package letterboxd

import (
	"strings"
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// TestOptionalMetadataValidation accepts unknowns and enforces database scalar boundaries.
func TestOptionalMetadataValidation(t *testing.T) {
	valid := entity.LetterboxdTitle{MediaType: "movie", TMDBID: 1, OriginalTitle: "Planned"}
	if err := ValidateTitle(&valid); err != nil {
		t.Fatal("unknown metadata", err)
	}
	for _, mutate := range []func(*entity.LetterboxdTitle){
		func(t *entity.LetterboxdTitle) { t.ReleaseYear = new(0) },
		func(t *entity.LetterboxdTitle) { t.ReleaseYear = new(-1) },
		func(t *entity.LetterboxdTitle) { t.ReleaseYear = new(10000) },
		func(t *entity.LetterboxdTitle) { t.RuntimeMinutes = new(0) },
		func(t *entity.LetterboxdTitle) { t.RuntimeMinutes = new(-1) },
		func(t *entity.LetterboxdTitle) { t.RuntimeMinutes = new(65536) },
		func(t *entity.LetterboxdTitle) { t.EpisodeCount = new(-1) },
		func(t *entity.LetterboxdTitle) { t.EpisodeCount = new(4294967296) },
		func(t *entity.LetterboxdTitle) { t.OriginalLanguage = new(strings.Repeat("a", 11)) },
		func(t *entity.LetterboxdTitle) { t.Status = new(strings.Repeat("a", 65)) },
		func(t *entity.LetterboxdTitle) { t.EpisodeRuntimeMinutes = []int{-1} },
	} {
		title := valid
		mutate(&title)
		if ValidateTitle(&title) == nil {
			t.Fatalf("invalid metadata accepted: %#v", title)
		}
	}
	for _, year := range []int{1, 9999} {
		title := valid
		title.ReleaseYear, title.RuntimeMinutes, title.EpisodeCount = new(year), new(65535), new(0)
		title.EpisodeRuntimeMinutes = []int{0, 45}
		if err := ValidateTitle(&title); err != nil {
			t.Fatal("valid boundary", err)
		}
	}
}
