package birthday

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	appbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/birthday"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

func TestRegisterAndListStartsInJanuary(t *testing.T) {
	service := &fakeService{birthdays: []*entity.Birthday{{
		Name: "Leo", Birthday: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
	}}}
	routes := discord.NewRoutes()
	Register(routes, service)
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	definitions, err := routes.Commands().Definitions()
	if err != nil || len(definitions) != 1 || definitions[0].Name != commandName || definitions[0].Description != ptbr.BirthdayCommandDescription {
		t.Fatalf("definitions = %#v, %v", definitions, err)
	}
	responder := &fakeResponder{}
	err = (Handler{service: service}).List(context.Background(), &command.CommandRequest{
		Actor: command.Actor{UserID: "123"}, Responder: responder,
	})
	if err != nil || service.month != time.January {
		t.Fatalf("List() error = %v, month = %v", err, service.month)
	}
	assertPage(t, responder.response, discordgo.InteractionResponseChannelMessageWithSource, "janeiro", "Leo")
}

func TestListReturnsServiceError(t *testing.T) {
	want := errors.New("list")
	err := (Handler{service: &fakeService{err: want}}).List(context.Background(), &command.CommandRequest{Responder: &fakeResponder{}})
	if !errors.Is(err, want) {
		t.Fatalf("List() error = %v", err)
	}
}

func TestChangePage(t *testing.T) {
	service := &fakeService{}
	handler := Handler{service: service}
	responder := &fakeResponder{}
	request := &discord.InteractionRequest{
		Actor:      command.Actor{UserID: "123"},
		Parameters: []string{"next", "1"}, Responder: responder,
	}
	if err := handler.ChangePage(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if service.month != time.February {
		t.Fatalf("month = %v", service.month)
	}
	assertPage(t, responder.response, discordgo.InteractionResponseUpdateMessage, "fevereiro", ptbr.BirthdayEmptyMonth)

	request.Parameters = []string{"previous", "12"}
	if err := handler.ChangePage(context.Background(), request); err != nil || service.month != time.November {
		t.Fatalf("previous error = %v, month = %v", err, service.month)
	}
	request.Parameters = []string{"next", "12"}
	if err := handler.ChangePage(context.Background(), request); err != nil || service.month != time.December {
		t.Fatalf("December boundary error = %v, month = %v", err, service.month)
	}
	request.Parameters = []string{"previous", "1"}
	if err := handler.ChangePage(context.Background(), request); err != nil || service.month != time.January {
		t.Fatalf("January boundary error = %v, month = %v", err, service.month)
	}
}

func TestChangePageIsAvailableToAnotherMember(t *testing.T) {
	service := &fakeService{}
	responder := &fakeResponder{}
	err := (Handler{service: service}).ChangePage(context.Background(), &discord.InteractionRequest{
		Actor: command.Actor{UserID: "456"}, Parameters: []string{"next", "1"}, Responder: responder,
	})
	if err != nil || service.month != time.February {
		t.Fatalf("ChangePage() error = %v, month = %v", err, service.month)
	}
}

func TestChangePageRejectsInvalidAndForeignButtons(t *testing.T) {
	handler := Handler{service: &fakeService{}}
	for name, parameters := range map[string][]string{
		"missing":   nil,
		"direction": {"sideways", "1", "123"},
		"month":     {"next", "13", "123"},
		"extra":     {"next", "1", "123"},
	} {
		t.Run(name, func(t *testing.T) {
			responder := &fakeResponder{}
			err := handler.ChangePage(context.Background(), &discord.InteractionRequest{
				Actor: command.Actor{UserID: "123"}, Parameters: parameters, Responder: responder,
			})
			if err != nil || responseText(responder.response) != ptbr.BirthdayInvalidInteraction {
				t.Fatalf("response = %#v, %v", responder.response, err)
			}
		})
	}
}

func TestChangePageReturnsServiceError(t *testing.T) {
	want := errors.New("month")
	err := (Handler{service: &fakeService{err: want}}).ChangePage(context.Background(), &discord.InteractionRequest{
		Actor: command.Actor{UserID: "123"}, Parameters: []string{"next", "1"}, Responder: &fakeResponder{},
	})
	if !errors.Is(err, want) {
		t.Fatalf("ChangePage() error = %v", err)
	}
}

func TestOpenModal(t *testing.T) {
	responder := &fakeResponder{}
	err := (Handler{}).OpenModal(context.Background(), &discord.InteractionRequest{
		Actor: command.Actor{Permissions: discordgo.PermissionManageGuild}, Responder: responder,
	})
	if err != nil || responder.response.Type != discordgo.InteractionResponseModal || responder.response.Data.Title != ptbr.BirthdayAddModalTitle || len(responder.response.Data.Components) != 4 {
		t.Fatalf("modal response = %#v, %v", responder.response, err)
	}
}

func TestBirthdayManagementRequiresManageServer(t *testing.T) {
	for name, handler := range map[string]func(*discord.InteractionRequest) error{
		"open modal": func(request *discord.InteractionRequest) error {
			return (Handler{}).OpenModal(context.Background(), request)
		},
		"submit modal": func(request *discord.InteractionRequest) error {
			return (Handler{}).Submit(context.Background(), request)
		},
	} {
		t.Run(name, func(t *testing.T) {
			responder := &fakeResponder{}
			request := &discord.InteractionRequest{Responder: responder}
			if name == "submit modal" {
				request.Interaction = &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
					Data: discordgo.ModalSubmitInteractionData{CustomID: addBirthdayRoute},
				}}
			}
			if err := handler(request); err != nil {
				t.Fatal(err)
			}
			if got := responseText(responder.response); got != ptbr.BirthdayManageServerRequired {
				t.Fatalf("response = %q", got)
			}
		})
	}
}

func TestAdministratorCanManageBirthdays(t *testing.T) {
	responder := &fakeResponder{}
	err := (Handler{}).OpenModal(context.Background(), &discord.InteractionRequest{
		Actor: command.Actor{Permissions: discordgo.PermissionAdministrator}, Responder: responder,
	})
	if err != nil || responder.response.Type != discordgo.InteractionResponseModal {
		t.Fatalf("response = %#v, error = %v", responder.response, err)
	}
}

func TestSubmitBirthday(t *testing.T) {
	service := &fakeService{}
	responder := &fakeResponder{}
	err := (Handler{service: service}).Submit(context.Background(), modalRequest("123", map[string]string{
		nameInputID: "Leo", birthdayInputID: "2000-03-04",
		timeZoneInputID: "America/Sao_Paulo", messageInputID: "Olá, {mention}",
	}, responder))
	if err != nil || responseText(responder.response) != ptbr.BirthdaySaved {
		t.Fatalf("Submit() response = %#v, %v", responder.response, err)
	}
	wantDate := time.Date(2000, 3, 4, 0, 0, 0, 0, time.UTC)
	if service.saved.UserID != 123 || !service.saved.Birthday.Equal(wantDate) || service.saved.Name != "Leo" {
		t.Fatalf("saved = %#v", service.saved)
	}
}

func TestSubmitValidation(t *testing.T) {
	tests := map[string]struct {
		userID string
		date   string
		err    error
		want   string
	}{
		"user":         {"bad", "2000-01-01", nil, ptbr.BirthdayInvalidInteraction},
		"date":         {"123", "01/01/2000", nil, ptbr.BirthdayInvalidDate},
		"name":         {"123", "2000-01-01", appbirthday.ErrInvalidName, ptbr.BirthdayInvalidName},
		"service date": {"123", "2000-01-01", appbirthday.ErrInvalidDate, ptbr.BirthdayInvalidDate},
		"zone":         {"123", "2000-01-01", appbirthday.ErrInvalidTimeZone, ptbr.BirthdayInvalidTimeZone},
		"message":      {"123", "2000-01-01", appbirthday.ErrInvalidMessage, ptbr.BirthdayInvalidMessage},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			responder := &fakeResponder{}
			err := (Handler{service: &fakeService{err: test.err}}).Submit(context.Background(), modalRequest(
				test.userID,
				map[string]string{birthdayInputID: test.date},
				responder,
			))
			if err != nil || responseText(responder.response) != test.want {
				t.Fatalf("response = %#v, %v", responder.response, err)
			}
		})
	}
	want := errors.New("database")
	err := (Handler{service: &fakeService{err: want}}).Submit(context.Background(), modalRequest(
		"123", map[string]string{birthdayInputID: "2000-01-01"}, &fakeResponder{},
	))
	if !errors.Is(err, want) {
		t.Fatalf("database error = %v", err)
	}
}

func TestModalValuesIgnoresUnknownComponents(t *testing.T) {
	components := []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: "ignored"},
		&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			&discordgo.TextInput{CustomID: "pointer", Value: "one"},
			discordgo.TextInput{CustomID: "value", Value: "two"},
		}},
	}
	want := map[string]string{"pointer": "one", "value": "two"}
	if got := modalValues(components); !reflect.DeepEqual(got, want) {
		t.Fatalf("modalValues() = %#v", got)
	}
}

func TestPageEscapesMarkdownAndDisablesMentions(t *testing.T) {
	response := pageResponse(discordgo.InteractionResponseChannelMessageWithSource, time.January, []*entity.Birthday{{
		Name: "*@everyone*", Birthday: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}})
	if got := responseText(response); !strings.Contains(got, "\\*@everyone\\*") {
		t.Fatalf("page content = %q", got)
	}
	if response.Data.AllowedMentions == nil || len(response.Data.AllowedMentions.Parse) != 0 {
		t.Fatalf("allowed mentions = %#v", response.Data.AllowedMentions)
	}
}

func TestSenderRendersAllowedMention(t *testing.T) {
	messenger := &fakeMessenger{}
	sender := NewSender(messenger, "channel")
	err := sender.Send(appbirthday.Announcement{
		UserID: 123, Name: "Leo", Age: 26,
		Message: "{mention}: {name} faz {age} anos!",
	})
	if err != nil {
		t.Fatal(err)
	}
	if messenger.channelID != "channel" || messenger.message.Flags != discordgo.MessageFlagsIsComponentsV2 || !reflect.DeepEqual(messenger.message.AllowedMentions.Users, []string{"123"}) {
		t.Fatalf("message = %#v", messenger.message)
	}
	container := messenger.message.Components[0].(discordgo.Container)
	text := container.Components[0].(discordgo.TextDisplay).Content
	if text != "<@123>: Leo faz 26 anos!" {
		t.Fatalf("content = %q", text)
	}
}

func TestSenderReturnsDeliveryError(t *testing.T) {
	want := errors.New("send")
	err := NewSender(&fakeMessenger{err: want}, "channel").Send(appbirthday.Announcement{})
	if !errors.Is(err, want) {
		t.Fatalf("Send() error = %v", err)
	}
}

func assertPage(t *testing.T, response *discordgo.InteractionResponse, responseType discordgo.InteractionResponseType, contents ...string) {
	t.Helper()
	if response == nil || response.Type != responseType || response.Data.Flags&discordgo.MessageFlagsIsComponentsV2 == 0 || len(response.Data.Components) != 2 {
		t.Fatalf("page response = %#v", response)
	}
	text := responseText(response)
	for _, content := range contents {
		if !strings.Contains(text, content) {
			t.Fatalf("page %q does not contain %q", text, content)
		}
	}
	row := response.Data.Components[1].(discordgo.ActionsRow)
	if len(row.Components) != 3 {
		t.Fatalf("button row = %#v", row)
	}
	for _, component := range row.Components[:2] {
		button := component.(discordgo.Button)
		if button.Label != "" || button.Emoji == nil {
			t.Fatalf("pagination button = %#v", button)
		}
	}
}

func responseText(response *discordgo.InteractionResponse) string {
	if response == nil || response.Data == nil || len(response.Data.Components) == 0 {
		return ""
	}
	switch component := response.Data.Components[0].(type) {
	case discordgo.TextDisplay:
		return component.Content
	case discordgo.Container:
		return component.Components[0].(discordgo.TextDisplay).Content
	default:
		return ""
	}
}

func modalRequest(userID string, values map[string]string, responder *fakeResponder) *discord.InteractionRequest {
	rows := make([]discordgo.MessageComponent, 0, len(values))
	for customID, value := range values {
		rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: customID, Value: value},
		}})
	}
	return &discord.InteractionRequest{
		Actor: command.Actor{UserID: command.Snowflake(userID), Permissions: discordgo.PermissionManageGuild}, Responder: responder,
		Interaction: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionModalSubmit,
			Data: discordgo.ModalSubmitInteractionData{CustomID: addBirthdayRoute, Components: rows},
		}},
	}
}

type fakeService struct {
	birthdays []*entity.Birthday
	month     time.Month
	saved     appbirthday.SaveInput
	err       error
}

func (s *fakeService) Month(month time.Month) ([]*entity.Birthday, error) {
	s.month = month
	return s.birthdays, s.err
}

func (s *fakeService) Save(input appbirthday.SaveInput) error {
	s.saved = input
	return s.err
}

type fakeResponder struct {
	response *discordgo.InteractionResponse
}

func (r *fakeResponder) Respond(response *discordgo.InteractionResponse) error {
	r.response = response
	return nil
}

type fakeMessenger struct {
	channelID string
	message   *discordgo.MessageSend
	err       error
}

func (m *fakeMessenger) SendMessage(channelID string, message *discordgo.MessageSend) error {
	m.channelID, m.message = channelID, message
	return m.err
}
