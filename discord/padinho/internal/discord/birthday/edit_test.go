package birthday

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func TestInspectShowsOnlyActorsFullRegistration(t *testing.T) {
	registration := registeredBirthday()
	service := &fakeService{birthday: registration}
	responder := &fakeResponder{}
	guild := &discordgo.Guild{ID: "guild", Name: "Salada", Icon: "icon"}
	lookup := &fakeGuildLookup{guild: guild}
	err := (Handler{service: service, guilds: lookup}).Inspect(context.Background(), &discord.InteractionRequest{
		Actor: command.Actor{UserID: "123"}, GuildID: "guild", Responder: responder,
	})
	if err != nil || service.birthdayUserID != 123 || lookup.guildID != "guild" {
		t.Fatalf("Inspect() error = %v, birthday lookup = %d, guild lookup = %q", err, service.birthdayUserID, lookup.guildID)
	}
	text := responseText(responder.response)
	for _, want := range []string{"04/03/2000", "`America/Sao_Paulo`", "Olá, \\*Leo\\*", "Salada"} {
		if !strings.Contains(text, want) {
			t.Fatalf("inspection %q does not contain %q", text, want)
		}
	}
	if responder.response.Data.Flags != discordgo.MessageFlagsEphemeral || len(responder.response.Data.Embeds) != 1 {
		t.Fatalf("inspection flags = %v", responder.response.Data.Flags)
	}
	footer := responder.response.Data.Embeds[0].Footer
	if footer == nil || footer.Text != guild.Name || footer.IconURL != guild.IconURL("64") {
		t.Fatalf("inspection footer = %#v", footer)
	}
}

func TestInspectHandlesDefaultMessageMissingRegistrationAndErrors(t *testing.T) {
	registration := registeredBirthday()
	registration.Message = ""
	responder := &fakeResponder{}
	if err := (Handler{service: &fakeService{birthday: registration}}).Inspect(context.Background(), &discord.InteractionRequest{
		Actor: command.Actor{UserID: "123"}, Responder: responder,
	}); err != nil || !strings.Contains(responseText(responder.response), ptbr.BirthdayDefaultMessageValue) || !strings.Contains(responseText(responder.response), ptbr.BirthdayGuildUnknown) {
		t.Fatalf("default inspection = %#v, %v", responder.response, err)
	}

	for name, test := range map[string]struct {
		actor string
		err   error
		want  string
	}{
		"invalid actor": {"bad", nil, ptbr.BirthdayInvalidInteraction},
		"missing":       {"123", appbirthday.ErrBirthdayNotFound, ptbr.BirthdaySelfNoRegistration},
	} {
		t.Run(name, func(t *testing.T) {
			responder := &fakeResponder{}
			err := (Handler{service: &fakeService{birthdayErr: test.err}}).Inspect(context.Background(), &discord.InteractionRequest{
				Actor: command.Actor{UserID: command.Snowflake(test.actor)}, Responder: responder,
			})
			if err != nil || responseText(responder.response) != test.want {
				t.Fatalf("Inspect() = %#v, %v", responder.response, err)
			}
		})
	}
	wantErr := errors.New("database")
	if err := (Handler{service: &fakeService{birthdayErr: wantErr}}).Inspect(context.Background(), &discord.InteractionRequest{
		Actor: command.Actor{UserID: "123"}, Responder: &fakeResponder{},
	}); !errors.Is(err, wantErr) {
		t.Fatalf("Inspect() error = %v", err)
	}
	guildErr := errors.New("guild")
	if err := (Handler{service: &fakeService{birthday: registration}, guilds: &fakeGuildLookup{err: guildErr}}).Inspect(context.Background(), &discord.InteractionRequest{
		Actor: command.Actor{UserID: "123"}, GuildID: "guild", Responder: &fakeResponder{},
	}); !errors.Is(err, guildErr) {
		t.Fatalf("Inspect() guild error = %v", err)
	}
}

func TestOpenDashboardRequiresAdministratorAndStartsEmpty(t *testing.T) {
	for name, permissions := range map[string]int64{
		"normal":        0,
		"manage server": discordgo.PermissionManageGuild,
	} {
		t.Run(name, func(t *testing.T) {
			responder := &fakeResponder{}
			err := (Handler{}).OpenDashboard(context.Background(), &discord.InteractionRequest{
				Actor: command.Actor{Permissions: permissions}, Responder: responder,
			})
			if err != nil || responseText(responder.response) != ptbr.BirthdayAdministratorRequired {
				t.Fatalf("OpenDashboard() = %#v, %v", responder.response, err)
			}
		})
	}

	responder := &fakeResponder{}
	err := (Handler{}).OpenDashboard(context.Background(), &discord.InteractionRequest{
		Actor: command.Actor{Permissions: discordgo.PermissionAdministrator}, Responder: responder,
	})
	if err != nil || strings.Count(responseText(responder.response), emptyDashboardValue) != 5 {
		t.Fatalf("dashboard = %#v, %v", responder.response, err)
	}
	if responder.response.Data.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Fatalf("dashboard flags = %v", responder.response.Data.Flags)
	}
	row := responder.response.Data.Components[1].(discordgo.ActionsRow)
	menu := row.Components[0].(discordgo.SelectMenu)
	if menu.CustomID != editSelectRoute || menu.MenuType != discordgo.UserSelectMenu || menu.MaxValues != 1 {
		t.Fatalf("dashboard select = %#v", menu)
	}
}

func TestSelectDashboardUserLoadsAndRendersEditableFields(t *testing.T) {
	service := &fakeService{birthday: registeredBirthday()}
	responder := &fakeResponder{}
	err := (Handler{service: service}).SelectDashboardUser(context.Background(), dashboardSelectRequest("123", responder))
	if err != nil || service.birthdayUserID != 123 {
		t.Fatalf("SelectDashboardUser() error = %v, lookup = %d", err, service.birthdayUserID)
	}
	if responder.response.Type != discordgo.InteractionResponseUpdateMessage || responder.response.Data.Flags&discordgo.MessageFlagsEphemeral != 0 {
		t.Fatalf("dashboard update = %#v", responder.response)
	}
	container := responder.response.Data.Components[0].(discordgo.Container)
	fields := container.Components[2:7]
	if fields[0].Type() != discordgo.TextDisplayComponent {
		t.Fatalf("user ID field = %#v", fields[0])
	}
	lastLabel := container.Components[len(container.Components)-1].(discordgo.TextDisplay)
	if lastLabel.Content != "### "+ptbr.BirthdayDashboardUserLabel {
		t.Fatalf("dashboard user label = %#v", lastLabel)
	}
	wantFields := []string{editFieldName, editFieldBirthday, editFieldTimeZone, editFieldMessage}
	for index, wantField := range wantFields {
		section := fields[index+1].(discordgo.Section)
		button := section.Accessory.(discordgo.Button)
		wantID := fmt.Sprintf("%s:%s:123", editFieldRoute, wantField)
		if button.CustomID != wantID {
			t.Fatalf("field button %d = %q, want %q", index, button.CustomID, wantID)
		}
	}
	menu := responder.response.Data.Components[1].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if len(menu.DefaultValues) != 1 || menu.DefaultValues[0].ID != "123" {
		t.Fatalf("selected dashboard user = %#v", menu.DefaultValues)
	}
}

func TestDashboardShowsDefaultMessageStateForEmptyStoredMessage(t *testing.T) {
	registration := registeredBirthday()
	registration.Message = ""
	response := dashboardResponse(discordgo.InteractionResponseUpdateMessage, registration.UserID, registration, "")
	if !strings.Contains(responseText(response), ptbr.BirthdayDefaultMessageValue) {
		t.Fatalf("dashboard text = %q", responseText(response))
	}
}

func TestSelectDashboardUserHandlesMissingInvalidUnauthorizedAndErrors(t *testing.T) {
	responder := &fakeResponder{}
	err := (Handler{service: &fakeService{birthdayErr: appbirthday.ErrBirthdayNotFound}}).SelectDashboardUser(
		context.Background(), dashboardSelectRequest("123", responder),
	)
	if err != nil || !strings.Contains(responseText(responder.response), ptbr.BirthdayNoRegistration) || responder.response.Type != discordgo.InteractionResponseUpdateMessage {
		t.Fatalf("missing dashboard = %#v, %v", responder.response, err)
	}

	for name, request := range map[string]*discord.InteractionRequest{
		"unauthorized":        dashboardSelectRequestWithPermissions("123", 0, &fakeResponder{}),
		"missing interaction": {Actor: command.Actor{Permissions: discordgo.PermissionAdministrator}, Responder: &fakeResponder{}},
		"invalid value":       dashboardSelectRequest("bad", &fakeResponder{}),
		"multiple values":     dashboardSelectRequestValues([]string{"123", "456"}, &fakeResponder{}),
	} {
		t.Run(name, func(t *testing.T) {
			if err := (Handler{service: &fakeService{}}).SelectDashboardUser(context.Background(), request); err != nil {
				t.Fatal(err)
			}
			want := ptbr.BirthdayInvalidInteraction
			if name == "unauthorized" {
				want = ptbr.BirthdayAdministratorRequired
			}
			if got := responseText(request.Responder.(*fakeResponder).response); got != want {
				t.Fatalf("response = %q, want %q", got, want)
			}
		})
	}
	wantErr := errors.New("find")
	if err := (Handler{service: &fakeService{birthdayErr: wantErr}}).SelectDashboardUser(
		context.Background(), dashboardSelectRequest("123", &fakeResponder{}),
	); !errors.Is(err, wantErr) {
		t.Fatalf("SelectDashboardUser() error = %v", err)
	}
}

func TestUpdateInputRejectsUnknownField(t *testing.T) {
	if _, err := updateInput("user_id", 123, "456"); err == nil {
		t.Fatal("updateInput() accepted immutable user ID")
	}
}

func TestBirthdayComponentResponsesSerialize(t *testing.T) {
	registration := registeredBirthday()
	responses := map[string]*discordgo.InteractionResponse{
		"page":       pageResponse(discordgo.InteractionResponseChannelMessageWithSource, time.March, []*entity.Birthday{registration}, nil),
		"inspection": inspectionResponse(registration, nil),
		"dashboard":  dashboardResponse(discordgo.InteractionResponseUpdateMessage, registration.UserID, registration, ""),
		"modal":      editModal(editFieldMessage, registration),
	}
	for name, response := range responses {
		t.Run(name, func(t *testing.T) {
			if _, err := json.Marshal(response); err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
		})
	}
}

type fakeGuildLookup struct {
	guild   *discordgo.Guild
	err     error
	guildID command.Snowflake
}

// Guild returns the configured guild fixture for birthday interaction tests.
func (l *fakeGuildLookup) Guild(guildID command.Snowflake) (*discordgo.Guild, error) {
	l.guildID = guildID
	return l.guild, l.err
}

func TestOpenEditModalPrefillsEachMutableField(t *testing.T) {
	registration := registeredBirthday()
	tests := map[string]struct {
		wantValue    string
		wantStyle    discordgo.TextInputStyle
		wantRequired bool
	}{
		editFieldName:     {registration.Name, discordgo.TextInputShort, true},
		editFieldBirthday: {"04/03/2000", discordgo.TextInputShort, true},
		editFieldTimeZone: {registration.TimeZone, discordgo.TextInputShort, true},
		editFieldMessage:  {registration.Message, discordgo.TextInputParagraph, false},
	}
	for field, test := range tests {
		t.Run(field, func(t *testing.T) {
			responder := &fakeResponder{}
			err := (Handler{service: &fakeService{birthday: registration}}).OpenEditModal(context.Background(), &discord.InteractionRequest{
				Actor:      command.Actor{Permissions: discordgo.PermissionAdministrator},
				Parameters: []string{field, "123"}, Responder: responder,
			})
			if err != nil || responder.response.Type != discordgo.InteractionResponseModal || responder.response.Data.CustomID != fmt.Sprintf("%s:%s:123", editSubmitRoute, field) {
				t.Fatalf("modal = %#v, %v", responder.response, err)
			}
			input := responder.response.Data.Components[0].(discordgo.Label).Component.(discordgo.TextInput)
			if input.Value != test.wantValue || input.Style != test.wantStyle || input.Required == nil || *input.Required != test.wantRequired {
				t.Fatalf("input = %#v", input)
			}
		})
	}
}

func TestOpenEditModalRejectsInvalidUnauthorizedAndStaleActions(t *testing.T) {
	for name, test := range map[string]struct {
		permissions int64
		parameters  []string
		service     *fakeService
		want        string
		wantType    discordgo.InteractionResponseType
	}{
		"unauthorized": {0, []string{editFieldName, "123"}, &fakeService{}, ptbr.BirthdayAdministratorRequired, discordgo.InteractionResponseChannelMessageWithSource},
		"field":        {discordgo.PermissionAdministrator, []string{"user_id", "123"}, &fakeService{}, ptbr.BirthdayInvalidInteraction, discordgo.InteractionResponseChannelMessageWithSource},
		"user":         {discordgo.PermissionAdministrator, []string{editFieldName, "bad"}, &fakeService{}, ptbr.BirthdayInvalidInteraction, discordgo.InteractionResponseChannelMessageWithSource},
		"missing":      {discordgo.PermissionAdministrator, []string{editFieldName, "123"}, &fakeService{birthdayErr: appbirthday.ErrBirthdayNotFound}, ptbr.BirthdayNoRegistration, discordgo.InteractionResponseUpdateMessage},
	} {
		t.Run(name, func(t *testing.T) {
			responder := &fakeResponder{}
			err := (Handler{service: test.service}).OpenEditModal(context.Background(), &discord.InteractionRequest{
				Actor: command.Actor{Permissions: test.permissions}, Parameters: test.parameters, Responder: responder,
			})
			if err != nil || responder.response.Type != test.wantType || !strings.Contains(responseText(responder.response), test.want) {
				t.Fatalf("response = %#v, %v", responder.response, err)
			}
		})
	}
	wantErr := errors.New("find")
	if err := (Handler{service: &fakeService{birthdayErr: wantErr}}).OpenEditModal(context.Background(), &discord.InteractionRequest{
		Actor: command.Actor{Permissions: discordgo.PermissionAdministrator}, Parameters: []string{editFieldName, "123"}, Responder: &fakeResponder{},
	}); !errors.Is(err, wantErr) {
		t.Fatalf("OpenEditModal() error = %v", err)
	}
}

func TestSubmitEditUpdatesEachFieldAndRefreshesDashboard(t *testing.T) {
	tests := map[string]struct {
		value string
		check func(appbirthday.UpdateInput) bool
	}{
		editFieldName: {"Leonardo", func(input appbirthday.UpdateInput) bool { return input.Name != nil && *input.Name == "Leonardo" }},
		editFieldBirthday: {"05/04/1999", func(input appbirthday.UpdateInput) bool {
			return input.Birthday != nil && input.Birthday.Equal(time.Date(1999, 4, 5, 0, 0, 0, 0, time.UTC))
		}},
		editFieldTimeZone: {"America/Manaus", func(input appbirthday.UpdateInput) bool {
			return input.TimeZone != nil && *input.TimeZone == "America/Manaus"
		}},
		editFieldMessage: {"Nova mensagem", func(input appbirthday.UpdateInput) bool {
			return input.Message != nil && *input.Message == "Nova mensagem"
		}},
	}
	for field, test := range tests {
		t.Run(field, func(t *testing.T) {
			service := &fakeService{birthday: registeredBirthday()}
			responder := &fakeResponder{}
			err := (Handler{service: service}).SubmitEdit(context.Background(), editModalRequest(field, "123", test.value, responder))
			if err != nil || !test.check(service.updated) || service.updated.UserID != 123 {
				t.Fatalf("SubmitEdit() error = %v, update = %#v", err, service.updated)
			}
			if responder.response.Type != discordgo.InteractionResponseUpdateMessage || !strings.Contains(responseText(responder.response), ptbr.BirthdayEditSaved) {
				t.Fatalf("dashboard refresh = %#v", responder.response)
			}
		})
	}
}

func TestSubmitEditHandlesInvalidValidationMissingAndRepositoryErrors(t *testing.T) {
	for name, test := range map[string]struct {
		request *discord.InteractionRequest
		service *fakeService
		want    string
	}{
		"permission": {editModalRequestWithPermissions(editFieldName, "123", "Leo", 0, &fakeResponder{}), &fakeService{}, ptbr.BirthdayAdministratorRequired},
		"parameters": {editModalRequest("user_id", "123", "Leo", &fakeResponder{}), &fakeService{}, ptbr.BirthdayInvalidInteraction},
		"date":       {editModalRequest(editFieldBirthday, "123", "1999-04-05", &fakeResponder{}), &fakeService{}, ptbr.BirthdayInvalidDate},
		"validation": {editModalRequest(editFieldMessage, "123", "bad", &fakeResponder{}), &fakeService{updateErr: appbirthday.ErrInvalidMessage}, ptbr.BirthdayInvalidMessage},
		"missing":    {editModalRequest(editFieldName, "123", "Leo", &fakeResponder{}), &fakeService{updateErr: appbirthday.ErrBirthdayNotFound}, ptbr.BirthdayNoRegistration},
	} {
		t.Run(name, func(t *testing.T) {
			if err := (Handler{service: test.service}).SubmitEdit(context.Background(), test.request); err != nil {
				t.Fatal(err)
			}
			if got := responseText(test.request.Responder.(*fakeResponder).response); !strings.Contains(got, test.want) {
				t.Fatalf("response = %q, want %q", got, test.want)
			}
		})
	}

	missingValue := editModalRequest(editFieldName, "123", "Leo", &fakeResponder{})
	missingValue.Interaction.Data = discordgo.ModalSubmitInteractionData{CustomID: editSubmitRoute}
	if err := (Handler{service: &fakeService{}}).SubmitEdit(context.Background(), missingValue); err != nil || responseText(missingValue.Responder.(*fakeResponder).response) != ptbr.BirthdayInvalidInteraction {
		t.Fatalf("missing value response = %#v, %v", missingValue.Responder.(*fakeResponder).response, err)
	}

	wantErr := errors.New("update")
	if err := (Handler{service: &fakeService{updateErr: wantErr}}).SubmitEdit(
		context.Background(), editModalRequest(editFieldName, "123", "Leo", &fakeResponder{}),
	); !errors.Is(err, wantErr) {
		t.Fatalf("SubmitEdit() error = %v", err)
	}
}

func registeredBirthday() *entity.Birthday {
	return &entity.Birthday{
		UserID: 123, Name: "Leo", Birthday: time.Date(2000, 3, 4, 0, 0, 0, 0, time.UTC),
		TimeZone: "America/Sao_Paulo", Message: "Olá, *Leo*",
	}
}

func dashboardSelectRequest(userID string, responder *fakeResponder) *discord.InteractionRequest {
	return dashboardSelectRequestWithPermissions(userID, discordgo.PermissionAdministrator, responder)
}

func dashboardSelectRequestWithPermissions(userID string, permissions int64, responder *fakeResponder) *discord.InteractionRequest {
	request := dashboardSelectRequestValues([]string{userID}, responder)
	request.Actor.Permissions = permissions
	return request
}

func dashboardSelectRequestValues(values []string, responder *fakeResponder) *discord.InteractionRequest {
	return &discord.InteractionRequest{
		Actor: command.Actor{Permissions: discordgo.PermissionAdministrator}, Responder: responder,
		Interaction: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			Data: discordgo.MessageComponentInteractionData{CustomID: editSelectRoute, Values: values},
		}},
	}
}

func editModalRequest(field, userID, value string, responder *fakeResponder) *discord.InteractionRequest {
	return editModalRequestWithPermissions(field, userID, value, discordgo.PermissionAdministrator, responder)
}

func editModalRequestWithPermissions(field, userID, value string, permissions int64, responder *fakeResponder) *discord.InteractionRequest {
	return &discord.InteractionRequest{
		Actor: command.Actor{Permissions: permissions}, Parameters: []string{field, userID}, Responder: responder,
		Interaction: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionModalSubmit,
			Data: discordgo.ModalSubmitInteractionData{
				CustomID: fmt.Sprintf("%s:%s:%s", editSubmitRoute, field, userID),
				Components: []discordgo.MessageComponent{discordgo.Label{
					Component: discordgo.TextInput{CustomID: editValueInputID, Value: value},
				}},
			},
		}},
	}
}
