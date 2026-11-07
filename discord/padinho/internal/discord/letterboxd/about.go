package letterboxd

import (
	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const tmdbLogoURL = "https://www.themoviedb.org/assets/apple-touch-icon-57ed4b3b0450fd5e9a0c20f34e814b82adaa1085c79bdde2f00ca8787b63d2c4.png"
const tmdbWebsiteURL = "https://www.themoviedb.org"

// aboutPage presents feature guidance and provider attribution entirely through Components V2.
func aboutPage() *discordgo.InteractionResponseData {
	divider := true
	logoDescription := ptbr.LetterboxdTMDBLogo
	attribution := discordgo.Section{
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: ptbr.LetterboxdAboutCredits + "\n" + ptbr.LetterboxdCredit + "\n[TMDB](" + tmdbWebsiteURL + ")"},
		},
		Accessory: discordgo.Thumbnail{
			Media:       discordgo.UnfurledMediaItem{URL: tmdbLogoURL},
			Description: &logoDescription,
		},
	}
	container := discordgo.Container{Components: []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: ptbr.LetterboxdAboutOverview},
		discordgo.TextDisplay{Content: ptbr.LetterboxdAboutLocal},
		discordgo.Separator{Divider: &divider},
		attribution,
	}}
	data := response("")
	data.Flags = discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2
	data.Components = []discordgo.MessageComponent{container}
	return data
}
