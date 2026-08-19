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

// NewSender constructs a Discord birthday announcement sender for a channel.
func NewSender(messenger Messenger, channelID string) *Sender {
	return &Sender{messenger: messenger, channelID: channelID}
}

// Send expands birthday placeholders and delivers the resulting announcement.
func (s *Sender) Send(announcement appbirthday.Announcement) error {
	userID := strconv.FormatUint(announcement.UserID, 10)
	mention := "<@" + userID + ">"
	content := strings.NewReplacer(
		"{age}", strconv.Itoa(announcement.Age),
		"{name}", announcement.Name,
		"{mention}", mention,
	).Replace(announcement.Message)
	payload := announcementMessage(content)

	return s.messenger.SendMessage(s.channelID, payload)
}

// announcementMessage builds a plain Discord message with default mention handling.
func announcementMessage(content string) *discordgo.MessageSend {
	return &discordgo.MessageSend{Content: content}
}
