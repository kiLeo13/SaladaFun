package letterboxd

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

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
			options = append(options, discordgo.SelectMenuOption{Label: clip(label(t), 100), Value: app.Identity(t.MediaType, t.TMDBID)})
		}
		data.Embeds = []*discordgo.MessageEmbed{{Description: clip(strings.Join(lines, "\n"), 3500)}}
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
		rating = fmt.Sprintf(ptbr.LetterboxdRating, float64(sum)/float64(len(opinions)), len(opinions))
	}
	embed := &discordgo.MessageEmbed{
		Title:       clip(label(*title), 256),
		Description: clip(title.Synopsis, 3000),
		Color:       0x5865F2,
		URL:         fmt.Sprintf("https://www.themoviedb.org/%s/%d", title.MediaType, title.TMDBID),
		Footer:      &discordgo.MessageEmbedFooter{Text: rating},
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
