package birthday

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	appbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/birthday"
)

// Messenger is the Discord delivery capability consumed by the announcer.
type Messenger interface {
	SendMessage(string, *discordgo.MessageSend) error
}

// Sender renders and delivers birthday announcements through Discord.
type Sender struct {
	messenger Messenger
	channelID string
}

func NewSender(messenger Messenger, channelID string) *Sender {
	return &Sender{messenger: messenger, channelID: channelID}
}

func (s *Sender) Send(announcement appbirthday.Announcement) error {
	userID := strconv.FormatUint(announcement.UserID, 10)
	mention := "<@" + userID + ">"
	content := strings.NewReplacer(
		"{age}", strconv.Itoa(announcement.Age),
		"{name}", announcement.Name,
		"{mention}", mention,
	).Replace(announcement.Message)
	payload := announcementMessage(content, userID)

	return s.messenger.SendMessage(s.channelID, payload)
}

func announcementMessage(content string, userID string) *discordgo.MessageSend {
	accent := birthdayAccentColor
	container := discordgo.Container{
		AccentColor: &accent,
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: content},
		},
	}

	return &discordgo.MessageSend{
		Flags:           discordgo.MessageFlagsIsComponentsV2,
		AllowedMentions: &discordgo.MessageAllowedMentions{Users: []string{userID}},
		Components:      []discordgo.MessageComponent{container},
	}
}
