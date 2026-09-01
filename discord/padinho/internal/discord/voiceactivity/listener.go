// Package voiceactivity publishes and records Discord voice channel transitions.
package voiceactivity

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	appvoiceactivity "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/voiceactivity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

const (
	joinColor  = 0x53E93B
	moveColor  = 0xFFFC25
	leaveColor = 0xFF2525
)

// Messenger sends voice activity embeds through Padinho's Discord identity.
type Messenger interface {
	SendMessage(string, *discordgo.MessageSend) error
}

// Recorder persists completed voice activity delivery attempts.
type Recorder interface {
	Record(appvoiceactivity.RecordInput) error
}

type activityKind string

const (
	activityJoin  activityKind = "join"
	activityMove  activityKind = "move"
	activityLeave activityKind = "leave"
)

// Listener subscribes to voice state updates and records each channel transition.
type Listener struct {
	channelID string
	messenger Messenger
	recorder  Recorder
	logger    *slog.Logger
	now       func() time.Time
}

// New constructs a voice activity listener for one configured log channel.
func New(channelID string, messenger Messenger, recorder Recorder, logger *slog.Logger) (*Listener, error) {
	if !validSnowflake(channelID) {
		return nil, errors.New("voice activity log channel ID is invalid")
	}
	if messenger == nil {
		return nil, errors.New("voice activity messenger is nil")
	}
	if recorder == nil {
		return nil, errors.New("voice activity recorder is nil")
	}
	if logger == nil {
		return nil, errors.New("voice activity logger is nil")
	}
	return &Listener{
		channelID: channelID, messenger: messenger, recorder: recorder,
		logger: logger, now: time.Now,
	}, nil
}

// Subscribe attaches the voice-state handler to Padinho's Discord session.
func (l *Listener) Subscribe(_ context.Context, session *discordgo.Session) {
	session.AddHandler(l.handleVoiceStateUpdate)
}

// handleVoiceStateUpdate delivers and records one channel transition, if present.
func (l *Listener) handleVoiceStateUpdate(session *discordgo.Session, event *discordgo.VoiceStateUpdate) {
	kind, oldChannelID, newChannelID, ok := classify(event)
	if !ok {
		return
	}
	input, err := recordInput(event, oldChannelID, newChannelID, l.now().UTC())
	if err != nil {
		l.logger.Error("voice activity IDs are invalid", "error", err)
		return
	}

	status := entity.VoiceActivityLogSent
	message, err := renderMessage(session, event, kind, oldChannelID, newChannelID, input.OccurredAt)
	if err == nil {
		err = l.messenger.SendMessage(l.channelID, message)
	}
	if err != nil {
		status = entity.VoiceActivityLogFailed
		l.logger.Error("voice activity publish failed", "guild_id", event.GuildID, "user_id", event.UserID, "error", err)
	}
	input.Status = status
	if err := l.recorder.Record(input); err != nil {
		l.logger.Error("voice activity persistence failed", "guild_id", event.GuildID, "user_id", event.UserID, "error", err)
	}
}

func classify(event *discordgo.VoiceStateUpdate) (activityKind, string, string, bool) {
	if event == nil || event.VoiceState == nil || event.GuildID == "" || event.UserID == "" {
		return "", "", "", false
	}
	oldChannelID := ""
	if event.BeforeUpdate != nil {
		oldChannelID = event.BeforeUpdate.ChannelID
	}
	newChannelID := event.ChannelID
	if oldChannelID == newChannelID || (oldChannelID == "" && newChannelID == "") {
		return "", "", "", false
	}
	if oldChannelID == "" {
		return activityJoin, oldChannelID, newChannelID, true
	}
	if newChannelID == "" {
		return activityLeave, oldChannelID, newChannelID, true
	}
	return activityMove, oldChannelID, newChannelID, true
}

func recordInput(event *discordgo.VoiceStateUpdate, oldChannelID, newChannelID string, occurredAt time.Time) (appvoiceactivity.RecordInput, error) {
	guildID, err := snowflake(event.GuildID)
	if err != nil {
		return appvoiceactivity.RecordInput{}, fmt.Errorf("guild ID: %w", err)
	}
	userID, err := snowflake(event.UserID)
	if err != nil {
		return appvoiceactivity.RecordInput{}, fmt.Errorf("user ID: %w", err)
	}
	oldID, err := optionalSnowflake(oldChannelID)
	if err != nil {
		return appvoiceactivity.RecordInput{}, fmt.Errorf("old channel ID: %w", err)
	}
	newID, err := optionalSnowflake(newChannelID)
	if err != nil {
		return appvoiceactivity.RecordInput{}, fmt.Errorf("new channel ID: %w", err)
	}
	return appvoiceactivity.RecordInput{
		GuildID: guildID, UserID: userID, OldChannelID: oldID,
		NewChannelID: newID, OccurredAt: occurredAt,
	}, nil
}

func renderMessage(session *discordgo.Session, event *discordgo.VoiceStateUpdate, kind activityKind, oldChannelID, newChannelID string, occurredAt time.Time) (*discordgo.MessageSend, error) {
	if session == nil || session.State == nil {
		return nil, errors.New("Discord session state is unavailable")
	}
	guild, err := session.State.Guild(event.GuildID)
	if err != nil {
		return nil, fmt.Errorf("read guild state: %w", err)
	}
	member, err := memberFor(session, event)
	if err != nil {
		return nil, err
	}
	oldChannel, err := channelFor(session, oldChannelID)
	if err != nil {
		return nil, err
	}
	newChannel, err := channelFor(session, newChannelID)
	if err != nil {
		return nil, err
	}
	name := member.DisplayName()
	if name == "" {
		return nil, errors.New("voice activity member display name is empty")
	}

	embed := &discordgo.MessageEmbed{
		Color: colorFor(kind), Timestamp: occurredAt.UTC().Format(time.RFC3339Nano),
		Author: &discordgo.MessageEmbedAuthor{Name: authorText(kind, name, oldChannel, newChannel), IconURL: member.AvatarURL("128")},
		Footer: &discordgo.MessageEmbedFooter{Text: guild.Name, IconURL: guild.IconURL("128")},
		Fields: fieldsFor(kind, event.UserID, oldChannelID, newChannelID, connectedCount(guild, channelIDFor(kind, oldChannelID, newChannelID))),
	}
	return &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}, AllowedMentions: &discordgo.MessageAllowedMentions{}}, nil
}

func memberFor(session *discordgo.Session, event *discordgo.VoiceStateUpdate) (*discordgo.Member, error) {
	if event.Member != nil && event.Member.User != nil {
		return event.Member, nil
	}
	member, err := session.State.Member(event.GuildID, event.UserID)
	if err != nil {
		return nil, fmt.Errorf("read member state: %w", err)
	}
	if member.User == nil {
		return nil, errors.New("voice activity member user is unavailable")
	}
	return member, nil
}

func channelFor(session *discordgo.Session, channelID string) (*discordgo.Channel, error) {
	if channelID == "" {
		return nil, nil
	}
	channel, err := session.State.Channel(channelID)
	if err != nil {
		return nil, fmt.Errorf("read voice channel state: %w", err)
	}
	if channel.Name == "" {
		return nil, errors.New("voice activity channel name is empty")
	}
	return channel, nil
}

func authorText(kind activityKind, memberName string, oldChannel, newChannel *discordgo.Channel) string {
	switch kind {
	case activityJoin:
		return memberName + " entrou em " + newChannel.Name
	case activityMove:
		return memberName + " foi para " + newChannel.Name
	case activityLeave:
		return memberName + " saiu de " + oldChannel.Name
	default:
		return ""
	}
}

func fieldsFor(kind activityKind, userID, oldChannelID, newChannelID string, connected int) []*discordgo.MessageEmbedField {
	member := &discordgo.MessageEmbedField{Name: "👥 Membro", Value: "<@" + userID + ">", Inline: true}
	switch kind {
	case activityMove:
		return []*discordgo.MessageEmbedField{member, {Name: "🔊 Canais", Value: "<#" + oldChannelID + "> -> <#" + newChannelID + ">", Inline: true}}
	default:
		channelID := channelIDFor(kind, oldChannelID, newChannelID)
		return []*discordgo.MessageEmbedField{member,
			{Name: "🎙 Conectados", Value: strconv.Itoa(connected), Inline: true},
			{Name: "🔊 Canal", Value: "<#" + channelID + ">", Inline: true},
		}
	}
}

func colorFor(kind activityKind) int {
	switch kind {
	case activityJoin:
		return joinColor
	case activityMove:
		return moveColor
	default:
		return leaveColor
	}
}

func channelIDFor(kind activityKind, oldChannelID, newChannelID string) string {
	if kind == activityLeave {
		return oldChannelID
	}
	return newChannelID
}

func connectedCount(guild *discordgo.Guild, channelID string) int {
	count := 0
	for _, state := range guild.VoiceStates {
		if state.ChannelID == channelID {
			count++
		}
	}
	return count
}

func optionalSnowflake(value string) (*uint64, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := snowflake(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func validSnowflake(value string) bool {
	_, err := snowflake(value)
	return err == nil
}

func snowflake(value string) (uint64, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, errors.New("must be a non-zero unsigned integer")
	}
	return parsed, nil
}
