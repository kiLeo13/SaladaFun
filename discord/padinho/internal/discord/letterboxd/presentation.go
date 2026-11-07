package letterboxd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const synopsisLimit = 2000

// resultPage renders already selected results without querying storage or TMDB.
func resultPage(action, query string, page int, items []entity.LetterboxdTitle, cached, more bool) *discordgo.InteractionResponseData {
	data := response("")
	if cached {
		data.Content = ptbr.LetterboxdCached
	}
	if len(items) == 0 {
		data.Content += "\n" + ptbr.LetterboxdEmpty
	} else {
		options := []discordgo.SelectMenuOption{}
		lines := []string{}
		seen := map[string]bool{}
		for _, t := range items {
			value := app.Identity(t.MediaType, t.TMDBID)
			if seen[value] || len(options) >= pageSize {
				continue
			}
			seen[value] = true
			lines = append(lines, label(t))
			options = append(options, discordgo.SelectMenuOption{Label: Ellipsis(label(t), 100), Value: app.Identity(t.MediaType, t.TMDBID)})
		}
		data.Embeds = []*discordgo.MessageEmbed{{Description: Ellipsis(strings.Join(lines, "\n"), 3500)}}
		target := action
		if action == "list" {
			target = "show"
		}
		selection := discordgo.SelectMenu{CustomID: "lb.select:" + target, Placeholder: ptbr.LetterboxdChoose, Options: options}
		data.Components = []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{selection}}}
	}
	if len(data.Embeds) == 0 {
		data.Embeds = []*discordgo.MessageEmbed{{Description: ptbr.LetterboxdEmpty}}
	}
	if query != "" {
		data.Embeds[0].Footer = &discordgo.MessageEmbedFooter{Text: query}
	}
	buttons := []discordgo.MessageComponent{}
	for _, nav := range []struct {
		label    string
		page     int
		disabled bool
	}{{ptbr.LetterboxdPrevious, page - 1, page == 1}, {ptbr.LetterboxdNext, page + 1, !more || page >= 500}} {
		buttons = append(buttons, discordgo.Button{Label: nav.label, Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("lb.page:%s:%d", action, nav.page), Disabled: nav.disabled})
	}
	data.Components = append(data.Components, discordgo.ActionsRow{Components: buttons})
	return data
}

// titleCard renders a cached title and one current opinion per community member.
func titleCard(title *entity.LetterboxdTitle, opinions []app.Opinion, value string) *discordgo.InteractionResponseData {
	data := response("")
	rating := ptbr.LetterboxdNoRatings
	if len(opinions) > 0 {
		sum := 0
		for _, o := range opinions {
			sum += o.Rating
		}
		average := strings.ReplaceAll(strings.TrimSuffix(fmt.Sprintf("%.1f", float64(sum)/float64(len(opinions))), ".0"), ".", ",")
		if len(opinions) == 1 {
			rating = fmt.Sprintf(ptbr.LetterboxdRatingSingle, average)
		} else {
			rating = fmt.Sprintf(ptbr.LetterboxdRating, average, len(opinions))
		}
	}
	embed := &discordgo.MessageEmbed{
		Title:       Ellipsis(app.DisplayTitle(title), 256),
		Description: Ellipsis(title.Synopsis, synopsisLimit),
		Color:       0x5865F2,
		URL:         fmt.Sprintf("https://www.themoviedb.org/%s/%d", title.MediaType, title.TMDBID),
		Footer:      &discordgo.MessageEmbedFooter{Text: rating},
	}
	homepage := app.HomepageURL(title.Homepage)
	if homepage == "" {
		homepage = embed.URL
	}
	kind := ptbr.LetterboxdMovie
	if title.MediaType == "tv" {
		kind = ptbr.LetterboxdTV
	}
	embed.Author = &discordgo.MessageEmbedAuthor{Name: kind, URL: homepage}
	release := strconv.Itoa(title.ReleaseYear)
	if title.MediaType == "tv" && title.SeasonCount > 0 {
		if title.SeasonCount == 1 {
			release = fmt.Sprintf(ptbr.LetterboxdReleaseSeason, title.ReleaseYear)
		} else {
			release = fmt.Sprintf(ptbr.LetterboxdReleaseSeasons, title.ReleaseYear, title.SeasonCount)
		}
	}
	embed.Fields = []*discordgo.MessageEmbedField{{Name: ptbr.LetterboxdReleaseYear, Value: release, Inline: true}}
	creatorLabel := ptbr.LetterboxdDirectors
	if title.MediaType == "tv" {
		creatorLabel = ptbr.LetterboxdCreators
	}
	for _, field := range []struct {
		name   string
		values []string
	}{
		{creatorLabel, title.Creators}, {ptbr.LetterboxdGenres, title.Genres},
	} {
		if names := app.MetadataNames(field.values); len(names) > 0 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: field.name, Value: Ellipsis(plainOpinion(strings.Join(names, ", ")), 600), Inline: true})
		}
	}
	if title.Accent != nil {
		embed.Color = int(*title.Accent)
	}
	if title.BackdropPath != "" {
		embed.Image = &discordgo.MessageEmbedImage{URL: "https://image.tmdb.org/t/p/w780" + title.BackdropPath}
	} else if title.PosterPath != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: "https://image.tmdb.org/t/p/w185" + title.PosterPath}
	}
	data.Embeds = []*discordgo.MessageEmbed{embed}
	reviewButton := discordgo.Button{Label: ptbr.LetterboxdReviews, Emoji: &discordgo.ComponentEmoji{Name: "\U0001f4ac"}, Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("lb.reviews:%d:1", title.ID)}
	wishlistButton := discordgo.Button{Label: ptbr.LetterboxdAdd, Emoji: &discordgo.ComponentEmoji{Name: "\u2795"}, Style: discordgo.PrimaryButton, CustomID: "lb.add:" + value}
	watchedButton := discordgo.Button{Label: ptbr.LetterboxdWatched, Emoji: &discordgo.ComponentEmoji{Name: "\u2705"}, Style: discordgo.SuccessButton, CustomID: fmt.Sprintf("lb.watch:%d", title.ID)}
	data.Components = []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{watchedButton, reviewButton, wishlistButton}},
	}
	return data
}
