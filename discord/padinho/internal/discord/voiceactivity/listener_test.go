package voiceactivity

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	appvoiceactivity "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/voiceactivity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

func TestRenderMessageMatchesVoiceActivityFormats(t *testing.T) {
	timestamp := time.Date(2026, time.September, 1, 3, 16, 17, 577000000, time.UTC)
	tests := map[string]struct {
		kind                       activityKind
		oldChannelID, newChannelID string
		voiceStates                []*discordgo.VoiceState
		wantAuthor                 string
		wantColor                  int
		wantFields                 []string
	}{
		"join": {
			kind: activityJoin, newChannelID: "30", wantAuthor: "_janjo entrou em 🍒・tabaco e country", wantColor: joinColor,
			voiceStates: []*discordgo.VoiceState{{UserID: "20", ChannelID: "30"}, {UserID: "21", ChannelID: "30"}},
			wantFields:  []string{"<@20>", "2", "<#30>"},
		},
		"move": {
			kind: activityMove, oldChannelID: "30", newChannelID: "31", wantAuthor: "_janjo foi para 🍻・toca da misturada", wantColor: moveColor,
			wantFields: []string{"<@20>", "<#30> -> <#31>"},
		},
		"leave": {
			kind: activityLeave, oldChannelID: "31", wantAuthor: "_janjo saiu de 🍻・toca da misturada", wantColor: leaveColor,
			voiceStates: []*discordgo.VoiceState{{UserID: "21", ChannelID: "31"}},
			wantFields:  []string{"<@20>", "1", "<#31>"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			session := testSession(t, test.voiceStates)
			event := testEvent(test.oldChannelID, test.newChannelID)
			message, err := renderMessage(session, event, test.kind, test.oldChannelID, test.newChannelID, timestamp)
			if err != nil || len(message.Embeds) != 1 {
				t.Fatalf("renderMessage() = %#v, %v", message, err)
			}
			embed := message.Embeds[0]
			if embed.Color != test.wantColor || embed.Author == nil || embed.Author.Name != test.wantAuthor || embed.Footer == nil || embed.Footer.Text != "🍒 Salada de Fruta" || embed.Timestamp != "2026-09-01T03:16:17.577Z" {
				t.Fatalf("embed = %#v", embed)
			}
			if len(embed.Fields) != len(test.wantFields) {
				t.Fatalf("fields = %#v", embed.Fields)
			}
			for index, value := range test.wantFields {
				if !embed.Fields[index].Inline || embed.Fields[index].Value != value {
					t.Fatalf("field %d = %#v, want value %q and inline", index, embed.Fields[index], value)
				}
			}
			if message.AllowedMentions == nil || len(message.AllowedMentions.Parse) != 0 {
				t.Fatalf("allowed mentions = %#v", message.AllowedMentions)
			}
		})
	}
}

func TestListenerRecordsSentAndFailedDelivery(t *testing.T) {
	tests := map[string]struct {
		messengerErr error
		wantStatus   entity.VoiceActivityLogStatus
	}{
		"sent":   {wantStatus: entity.VoiceActivityLogSent},
		"failed": {messengerErr: errors.New("forbidden"), wantStatus: entity.VoiceActivityLogFailed},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			messenger := &fakeMessenger{err: test.messengerErr}
			recorder := &fakeRecorder{}
			listener, err := New("99", messenger, recorder, discardLogger())
			if err != nil {
				t.Fatal(err)
			}
			listener.now = func() time.Time { return time.Date(2026, time.September, 1, 3, 16, 17, 0, time.UTC) }
			listener.handleVoiceStateUpdate(testSession(t, []*discordgo.VoiceState{{UserID: "20", ChannelID: "30"}}), testEvent("", "30"))
			if messenger.messages != 1 || recorder.input == nil || recorder.input.Status != test.wantStatus || recorder.input.GuildID != 10 || recorder.input.UserID != 20 || recorder.input.NewChannelID == nil || *recorder.input.NewChannelID != 30 {
				t.Fatalf("delivery = messages:%d input:%#v", messenger.messages, recorder.input)
			}
		})
	}
}

func TestListenerIgnoresNonTransitionsAndRecordsRenderingFailure(t *testing.T) {
	messenger := &fakeMessenger{}
	recorder := &fakeRecorder{}
	listener, err := New("99", messenger, recorder, discardLogger())
	if err != nil {
		t.Fatal(err)
	}
	session := testSession(t, nil)
	unchanged := testEvent("30", "30")
	listener.handleVoiceStateUpdate(session, unchanged)
	if messenger.messages != 0 || recorder.input != nil {
		t.Fatalf("unchanged update was handled: messages:%d input:%#v", messenger.messages, recorder.input)
	}

	missingChannel := testEvent("", "999")
	listener.handleVoiceStateUpdate(session, missingChannel)
	if messenger.messages != 0 || recorder.input == nil || recorder.input.Status != entity.VoiceActivityLogFailed {
		t.Fatalf("rendering failure = messages:%d input:%#v", messenger.messages, recorder.input)
	}
}

func TestNewRejectsInvalidDependencies(t *testing.T) {
	validMessenger, validRecorder, logger := &fakeMessenger{}, &fakeRecorder{}, discardLogger()
	tests := []struct {
		channelID string
		messenger Messenger
		recorder  Recorder
		logger    *slog.Logger
	}{
		{"", validMessenger, validRecorder, logger}, {"zero", validMessenger, validRecorder, logger}, {"99", nil, validRecorder, logger}, {"99", validMessenger, nil, logger}, {"99", validMessenger, validRecorder, nil},
	}
	for _, test := range tests {
		if _, err := New(test.channelID, test.messenger, test.recorder, test.logger); err == nil {
			t.Fatal("New() error = nil")
		}
	}
}

func testSession(t *testing.T, voiceStates []*discordgo.VoiceState) *discordgo.Session {
	t.Helper()
	session := &discordgo.Session{State: discordgo.NewState()}
	guild := &discordgo.Guild{
		ID: "10", Name: "🍒 Salada de Fruta", Icon: "guild-icon", VoiceStates: voiceStates,
		Channels: []*discordgo.Channel{
			{ID: "30", GuildID: "10", Name: "🍒・tabaco e country", Type: discordgo.ChannelTypeGuildVoice},
			{ID: "31", GuildID: "10", Name: "🍻・toca da misturada", Type: discordgo.ChannelTypeGuildVoice},
		},
	}
	if err := session.State.GuildAdd(guild); err != nil {
		t.Fatal(err)
	}
	return session
}

func testEvent(oldChannelID, newChannelID string) *discordgo.VoiceStateUpdate {
	return &discordgo.VoiceStateUpdate{
		VoiceState: &discordgo.VoiceState{GuildID: "10", UserID: "20", ChannelID: newChannelID, Member: &discordgo.Member{
			GuildID: "10", Nick: "_janjo", User: &discordgo.User{ID: "20", Username: "janjo", Avatar: "avatar"},
		}},
		BeforeUpdate: &discordgo.VoiceState{GuildID: "10", UserID: "20", ChannelID: oldChannelID},
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeMessenger struct {
	messages int
	err      error
}

func (m *fakeMessenger) SendMessage(string, *discordgo.MessageSend) error {
	m.messages++
	return m.err
}

type fakeRecorder struct {
	input *appvoiceactivity.RecordInput
	err   error
}

func (r *fakeRecorder) Record(input appvoiceactivity.RecordInput) error {
	r.input = &input
	return r.err
}
