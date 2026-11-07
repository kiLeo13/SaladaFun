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
		text = app.NormalizeReview(*state.Review)
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
	if err := deferReply(r.Responder, r.GuildID, false); err != nil {
		return err
	}
	return h.renderHistory(ctx, r.Responder, user, sort, order, 1, historyAuthor(user, r.Users))
}

// historyPage validates navigation and retains the requested member and ordering.
func (h Handler) historyPage(ctx context.Context, r *discord.InteractionRequest) error {
	if _, err := component(r, false); err != nil {
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
	author := historyAuthor(user, nil)
	if r.Interaction != nil && r.Interaction.Message != nil && len(r.Interaction.Message.Embeds) > 0 {
		previous := r.Interaction.Message.Embeds[0].Author
		if previous != nil && previous.URL == author.URL {
			copy := *previous
			author = &copy
		}
	}
	return h.renderHistory(ctx, r.Responder, user, r.Parameters[1], r.Parameters[2], page, author)
}

// plainOpinion escapes Markdown and preserves readable short reviews.
func plainOpinion(s string) string {
	s = app.NormalizeReview(s)
	return strings.NewReplacer("\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "~", "\\~", "|", "\\|", ">", "\\>").Replace(s)
}

// renderHistory renders up to three compact entries in one public history embed.
func (h Handler) renderHistory(ctx context.Context, r command.Responder, user uint64, sort, order string, page int, authors ...*discordgo.MessageEmbedAuthor) error {
	results, err := h.service.Activity.History(ctx, user, sort, order, page)
	if errors.Is(err, app.ErrInvalid) {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	if err != nil {
		return err
	}
	entries := results.Entries
	if len(entries) > app.HistoryPageSize {
		entries = entries[:app.HistoryPageSize]
	}
	more := page*app.HistoryPageSize < results.Total
	author := historyAuthor(user, nil)
	if len(authors) > 0 && authors[0] != nil {
		author = authors[0]
	}
	lines := []string{}
	for _, e := range entries {
		title := entity.LetterboxdTitle{OriginalTitle: e.OriginalTitle, LocalizedTitle: e.LocalizedTitle}
		text := ptbr.LetterboxdReviewAbsent
		if e.Review != nil {
			if review := plainOpinion(*e.Review); review != "" {
				text = review
			}
		}
		lines = append(lines, fmt.Sprintf(ptbr.LetterboxdHistoryEntry, Ellipsis(plainOpinion(app.DisplayTitle(&title)), 300), e.Rating, time.UnixMilli(e.WatchedAt).Unix(), text))
	}
	description := strings.Join(lines, "\n\n")
	if len(entries) == 0 {
		description = ptbr.LetterboxdNoHistory
	}
	total := fmt.Sprintf(ptbr.LetterboxdReviewTotal, results.Total)
	if results.Total == 1 {
		total = ptbr.LetterboxdReviewTotalSingle
	}
	data := response("")
	data.Embeds = []*discordgo.MessageEmbed{{Author: author, Description: description, Footer: &discordgo.MessageEmbedFooter{Text: total}}}
	buttons := []discordgo.MessageComponent{}
	for _, n := range []struct {
		label, emoji string
		page         int
		disabled     bool
	}{{ptbr.LetterboxdPrevious, "\u2b05\ufe0f", page - 1, page == 1}, {ptbr.LetterboxdNext, "\u27a1\ufe0f", page + 1, !more || page >= 100000}} {
		buttons = append(buttons, discordgo.Button{Label: n.label, Emoji: &discordgo.ComponentEmoji{Name: n.emoji}, Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("lb.history:%d:%s:%s:%d", user, sort, order, n.page), Disabled: n.disabled})
	}
	data.Components = []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
	return finish(r, data)
}

// historyAuthor uses interaction profile data and falls back to the member snowflake.
func historyAuthor(user uint64, profiles map[command.Snowflake]command.UserProfile) *discordgo.MessageEmbedAuthor {
	id := strconv.FormatUint(user, 10)
	profile := profiles[command.Snowflake(id)]
	name := strings.TrimSpace(profile.GlobalName)
	if name == "" {
		name = strings.TrimSpace(profile.Username)
	}
	if name == "" {
		name = id
	}
	return &discordgo.MessageEmbedAuthor{Name: Ellipsis(fmt.Sprintf(ptbr.LetterboxdHistoryTitle, app.NormalizeReview(name)), 256), URL: "https://discord.com/users/" + id, IconURL: profile.AvatarURL}
}
