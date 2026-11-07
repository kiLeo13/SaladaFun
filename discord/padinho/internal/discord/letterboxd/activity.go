package letterboxd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// actorTitle validates a modal/button target independently of client-provided ownership.
func actorTitle(r *discord.InteractionRequest) (uint64, uint64, error) {
	if r.GuildID == "" {
		return 0, 0, command.RejectForbidden(ptbr.LetterboxdGuild)
	}
	if len(r.Parameters) != 1 {
		return 0, 0, command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	user, err := strconv.ParseUint(string(r.Actor.UserID), 10, 64)
	title, e := strconv.ParseUint(r.Parameters[0], 10, 64)
	if err != nil || e != nil || user == 0 || title == 0 {
		return 0, 0, command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	return user, title, nil
}

// openWatch opens a prefilled modal using local data only; cancellation writes nothing.
func (h Handler) openWatch(ctx context.Context, r *discord.InteractionRequest) error {
	user, title, err := actorTitle(r)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	state, err := h.service.Activity.State(ctx, user, title)
	if err != nil {
		return err
	}
	options := []discordgo.SelectMenuOption{}
	for rating := 1; rating <= 5; rating++ {
		value := strconv.Itoa(rating)
		options = append(options, discordgo.SelectMenuOption{Label: value, Value: value, Default: state.Rating == rating})
	}
	required := true
	text := ""
	if state.Review != nil {
		text = *state.Review
	}
	fields := []discordgo.MessageComponent{
		discordgo.Label{Label: ptbr.LetterboxdStars, Component: discordgo.SelectMenu{MenuType: discordgo.StringSelectMenu, CustomID: "rating", Options: options, MinValues: new(1), MaxValues: 1, Required: &required}},
		discordgo.Label{Label: ptbr.LetterboxdReview, Description: ptbr.LetterboxdReviewHint, Component: discordgo.TextInput{CustomID: "review", Style: discordgo.TextInputParagraph, MinLength: 2, MaxLength: 500, Required: new(false), Value: text}},
	}
	return r.Responder.Respond(&discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &discordgo.InteractionResponseData{Title: ptbr.LetterboxdModal, CustomID: fmt.Sprintf("lb.watch-save:%d", title), Components: fields}})
}

// modalFields collects Label/legacy row values while rejecting duplicates and unknown fields.
func modalFields(components []discordgo.MessageComponent) (int, string, error) {
	rating := 0
	text := ""
	seen := map[string]bool{}
	valid := true
	var visit func(discordgo.MessageComponent)
	field := func(id string) {
		if seen[id] {
			valid = false
		}
		seen[id] = true
	}
	visit = func(c discordgo.MessageComponent) {
		switch v := c.(type) {
		case *discordgo.Label:
			visit(v.Component)
		case discordgo.Label:
			visit(v.Component)
		case *discordgo.ActionsRow:
			for _, child := range v.Components {
				visit(child)
			}
		case discordgo.ActionsRow:
			for _, child := range v.Components {
				visit(child)
			}
		case *discordgo.SelectMenu:
			visit(*v)
		case discordgo.SelectMenu:
			field(v.CustomID)
			if v.CustomID != "rating" || len(v.Values) != 1 {
				valid = false
				return
			}
			var err error
			rating, err = strconv.Atoi(v.Values[0])
			if err != nil {
				valid = false
			}
		case *discordgo.TextInput:
			visit(*v)
		case discordgo.TextInput:
			field(v.CustomID)
			if v.CustomID != "review" {
				valid = false
				return
			}
			text = v.Value
		default:
			valid = false
		}
	}
	for _, c := range components {
		visit(c)
	}
	if !valid || !seen["rating"] || rating < 1 || rating > 5 {
		return 0, "", app.ErrInvalid
	}
	if _, err := app.ReviewText(text); err != nil {
		return 0, "", err
	}
	return rating, text, nil
}

// saveWatch commits the actor's opinion only after a valid modal submission.
func (h Handler) saveWatch(ctx context.Context, r *discord.InteractionRequest) error {
	user, title, err := actorTitle(r)
	if err != nil {
		return err
	}
	if r.Interaction == nil || r.Interaction.Type != discordgo.InteractionModalSubmit {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	rating, text, err := modalFields(r.Interaction.ModalSubmitData().Components)
	if err != nil {
		return command.RejectForbidden(ptbr.LetterboxdInvalidReview)
	}
	mutation, err := strconv.ParseUint(r.Interaction.ID, 10, 64)
	if err != nil || mutation == 0 {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	if err := deferReply(r.Responder, r.GuildID); err != nil {
		return err
	}
	if err := h.service.Activity.Save(ctx, user, title, mutation, rating, text); err != nil {
		return err
	}
	return finish(r.Responder, response(ptbr.LetterboxdSaved))
}

// pickRandom draws locally and lets metadata refresh fall back to the cached title.
func (h Handler) pickRandom(ctx context.Context, r *command.CommandRequest) error {
	user, err := strconv.ParseUint(string(r.Actor.UserID), 10, 64)
	if err != nil || user == 0 {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	if err := deferReply(r.Responder, r.GuildID); err != nil {
		return err
	}
	title, err := h.service.Activity.Random(ctx, user)
	if errors.Is(err, app.ErrEmptyWishlist) {
		return finish(r.Responder, response(ptbr.LetterboxdNoRandom))
	}
	if err != nil {
		return err
	}
	return h.showSelection(ctx, r.Responder, user, "show", app.Identity(title.MediaType, title.TMDBID))
}

// history defaults to the caller, watched date, and descending order.
func (h Handler) history(ctx context.Context, r *command.CommandRequest) error {
	member := r.Actor.UserID
	if option, err := r.Options.Snowflake("member"); err == nil {
		member = option
	}
	user, err := strconv.ParseUint(string(member), 10, 64)
	if err != nil || user == 0 {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	sort := "date"
	order := "desc"
	if value, err := r.Options.String("sort"); err == nil {
		sort = value
	}
	if value, err := r.Options.String("order"); err == nil {
		order = value
	}
	if err := deferReply(r.Responder, r.GuildID); err != nil {
		return err
	}
	return h.renderHistory(ctx, r.Responder, user, sort, order, 1)
}

// historyPage validates navigation and retains the requested member and ordering.
func (h Handler) historyPage(ctx context.Context, r *discord.InteractionRequest) error {
	if _, err := component(r); err != nil {
		return err
	}
	if len(r.Parameters) != 4 {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	user, err := strconv.ParseUint(r.Parameters[0], 10, 64)
	page, e := strconv.Atoi(r.Parameters[3])
	if err != nil || e != nil {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	return h.renderHistory(ctx, r.Responder, user, r.Parameters[1], r.Parameters[2], page)
}

// plainOpinion escapes Markdown and preserves readable short reviews.
func plainOpinion(s string) string {
	return strings.NewReplacer("\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "~", "\\~", "|", "\\|", ">", "\\>").Replace(s)
}

// renderHistory shows complete reviews in bounded embeds, with stable pagination.
func (h Handler) renderHistory(ctx context.Context, r command.Responder, user uint64, sort, order string, page int) error {
	entries, err := h.service.Activity.History(ctx, user, sort, order, page)
	if errors.Is(err, app.ErrInvalid) {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	if err != nil {
		return err
	}
	more := len(entries) > app.HistoryPageSize
	if more {
		entries = entries[:app.HistoryPageSize]
	}
	data := response(fmt.Sprintf(ptbr.LetterboxdHistoryTitle, user))
	if len(entries) == 0 {
		data.Content += "\n" + ptbr.LetterboxdNoHistory
	}
	for _, e := range entries {
		title := entity.LetterboxdTitle{OriginalTitle: e.OriginalTitle, LocalizedTitle: e.LocalizedTitle, MediaType: e.MediaType, ReleaseYear: e.ReleaseYear}
		text := ""
		if e.Review != nil {
			text = plainOpinion(*e.Review)
		}
		data.Embeds = append(data.Embeds, &discordgo.MessageEmbed{Description: fmt.Sprintf(ptbr.LetterboxdHistoryEntry, clip(plainOpinion(label(title)), 300), time.UnixMilli(e.WatchedAt).UTC().Format("02/01/2006 15:04"), e.Rating, text)})
	}
	buttons := []discordgo.MessageComponent{}
	for _, n := range []struct {
		label, emoji string
		page         int
		disabled     bool
	}{{ptbr.LetterboxdPrevious, "\u2b05\ufe0f", page - 1, page == 1}, {ptbr.LetterboxdNext, "\u27a1\ufe0f", page + 1, !more}} {
		buttons = append(buttons, discordgo.Button{Label: n.label, Emoji: &discordgo.ComponentEmoji{Name: n.emoji}, Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("lb.history:%d:%s:%s:%d", user, sort, order, n.page), Disabled: n.disabled})
	}
	data.Components = []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
	return finish(r, data)
}
