// Package letterboxd owns movie discovery and personal wishlist interactions.
package letterboxd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const pageSize = 20
const reviewPageSize = 3

// Handler keeps all discovery interactions on one application service.
type Handler struct{ service *app.Discovery }

// Register declares commands and restart-safe component routes.
func Register(routes *discord.Routes, service *app.Discovery) {
	h := Handler{service}
	group := routes.Commands().Group("letterboxd", ptbr.LetterboxdDescription)
	option := func() command.Option {
		return command.StringOption("title", ptbr.LetterboxdTitleOption).Required().Autocomplete()
	}
	group.Sub("show", ptbr.LetterboxdShow, h.run("show"), option())
	group.Sub("pick-random", ptbr.LetterboxdRandom, h.pickRandom)
	memberOption := command.UserOption("member", ptbr.LetterboxdMember)
	sortOption := command.StringOption("sort", ptbr.LetterboxdSort).Choices(command.OptionChoice{Name: ptbr.LetterboxdDate, Value: "date"}, command.OptionChoice{Name: ptbr.LetterboxdRatings, Value: "rating"})
	orderOption := command.StringOption("order", ptbr.LetterboxdOrder).Choices(command.OptionChoice{Name: ptbr.LetterboxdAscending, Value: "asc"}, command.OptionChoice{Name: ptbr.LetterboxdDescending, Value: "desc"})
	group.Sub("history", ptbr.LetterboxdHistory, h.history, memberOption, sortOption, orderOption)
	group.Sub("about", ptbr.LetterboxdAbout, h.about)
	wishlist := group.Group("wishlist", ptbr.LetterboxdWishlist)
	wishlist.Sub("add", ptbr.LetterboxdAdd, h.run("add"), option())
	wishlist.Sub("remove", ptbr.LetterboxdRemove, h.run("remove"), option())
	wishlist.Sub("list", ptbr.LetterboxdList, h.run("list"))
	for _, action := range []string{"show", "add", "remove"} {
		key := "letterboxd." + action + ".title"
		if action != "show" {
			key = "letterboxd.wishlist." + action + ".title"
		}
		routes.Autocomplete(key, h.autocomplete(action))
	}
	routes.Component("lb.select", h.selectTitle)
	routes.Component("lb.page", h.changePage)
	routes.Component("lb.add", h.add)
	routes.Component("lb.reviews", h.reviews)
	routes.Component("lb.watch", h.openWatch)
	routes.Modal("lb.watch-save", h.saveWatch)
	routes.Component("lb.history", h.historyPage)
}

// response builds ordinary embeds/components with mention suppression.
func response(text string) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{Content: text, AllowedMentions: &discordgo.MessageAllowedMentions{}}
}

// finish completes an acknowledged interaction via its bound transport.
func finish(r command.Responder, data *discordgo.InteractionResponseData) error {
	c, ok := r.(interface {
		Complete(*discordgo.InteractionResponseData) error
	})
	if !ok {
		return fmt.Errorf("responder cannot complete deferred interaction")
	}
	return c.Complete(data)
}

// deferReply acknowledges before provider/database work.
func deferReply(r command.Responder, guild command.Snowflake, private ...bool) error {
	if guild == "" {
		return command.RejectForbidden(ptbr.LetterboxdGuild)
	}
	flags := discordgo.MessageFlagsEphemeral
	if len(private) > 0 && !private[0] {
		flags = 0
	}
	return r.Respond(&discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: flags}})
}

// run creates a command executor for a fixed, registered action.
func (h Handler) run(action string) command.HandlerFunc {
	return func(ctx context.Context, r *command.CommandRequest) error {
		if err := deferReply(r.Responder, r.GuildID, action != "show"); err != nil {
			return err
		}
		user, err := strconv.ParseUint(string(r.Actor.UserID), 10, 64)
		if err != nil || user == 0 {
			return command.RejectForbidden(ptbr.LetterboxdInvalid)
		}
		query, _ := r.Options.String("title")
		if _, _, err := app.ParseIdentity(query); err == nil {
			return h.showSelection(ctx, r.Responder, user, action, query)
		}
		return h.page(ctx, r.Responder, user, action, query, 1)
	}
}

// autocomplete answers partial user input without writing metadata.
func (h Handler) autocomplete(action string) discord.AutocompleteHandler {
	return func(ctx context.Context, r *discord.AutocompleteRequest) error {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		choices := []*discordgo.ApplicationCommandOptionChoice{}
		query, _ := r.Options.String(r.Focused)
		user, _ := strconv.ParseUint(string(r.Actor.UserID), 10, 64)
		if r.GuildID != "" && user > 0 && utf8.RuneCountInString(query) <= 200 {
			items, _, err := h.service.Search(ctx, user, action, query, 1)
			if err == nil {
				for _, t := range items {
					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: clip(label(t), 100), Value: app.Identity(t.MediaType, t.TMDBID)})
					if len(choices) == 25 {
						break
					}
				}
			}
		}
		return r.Responder.Respond(&discordgo.InteractionResponse{Type: discordgo.InteractionApplicationCommandAutocompleteResult, Data: &discordgo.InteractionResponseData{Choices: choices}})
	}
}

// label disambiguates localized titles with media type and year.
func label(t entity.LetterboxdTitle) string {
	kind := ptbr.LetterboxdMovie
	if t.MediaType == "tv" {
		kind = ptbr.LetterboxdTV
	}
	year := "?"
	if t.ReleaseYear > 0 {
		year = strconv.Itoa(t.ReleaseYear)
	}
	return fmt.Sprintf("%s (%s, %s)", app.DisplayTitle(&t), year, kind)
}

// clip bounds Discord payload strings without splitting UTF-8 characters.
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n-1]) + "\u2026"
	}
	return s
}

// page lists remote matches or locally stored interests with reconstructible navigation.
func (h Handler) page(ctx context.Context, r command.Responder, user uint64, action, query string, page int) error {
	if utf8.RuneCountInString(query) > 200 || page < 1 || page > 500 {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	items, cached, err := h.service.Search(ctx, user, action, query, page)
	if err != nil {
		return err
	}
	if action == "show" && len(items) == 1 {
		return h.showSelection(ctx, r, user, action, app.Identity(items[0].MediaType, items[0].TMDBID))
	}
	more := len(items) > 0
	if action == "remove" || action == "list" || cached {
		start := (page - 1) * pageSize
		if start >= len(items) {
			more = false
			items = nil
		} else {
			more = len(items) > start+pageSize
			items = items[start:]
		}
	}
	if len(items) > pageSize {
		items = items[:pageSize]
	}
	return finish(r, resultPage(action, query, page, items, cached, more))
}

// showSelection performs the explicit selected-title action and renders its result.
func (h Handler) showSelection(ctx context.Context, r command.Responder, user uint64, action, value string) error {
	title, stale, err := h.service.Select(ctx, user, action, value)
	if errors.Is(err, app.ErrWishlistAbsent) {
		return finish(r, response(ptbr.LetterboxdAbsent))
	}
	if err != nil {
		return err
	}
	data := response("")
	if action == "remove" {
		return finish(r, response(ptbr.LetterboxdRemoved))
	}
	if action == "add" {
		data.Content = ptbr.LetterboxdAdded
	}
	if stale {
		data.Content += "\n" + ptbr.LetterboxdCached
	}
	opinions, err := h.service.Catalog.Reviews(ctx, title.ID)
	if err != nil {
		return err
	}
	card := titleCard(title, opinions, value)
	card.Content = data.Content
	return finish(r, card)
}

// component validates actor context and acknowledges a button/select interaction.
func component(r *discord.InteractionRequest) (uint64, error) {
	if err := deferReply(r.Responder, r.GuildID); err != nil {
		return 0, err
	}
	id, err := strconv.ParseUint(string(r.Actor.UserID), 10, 64)
	if err != nil || id == 0 {
		return 0, command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	return id, nil
}

// selectTitle resolves one selected remote identity.
func (h Handler) selectTitle(ctx context.Context, r *discord.InteractionRequest) error {
	user, err := component(r)
	if err != nil {
		return err
	}
	if len(r.Parameters) != 1 || !validAction(r.Parameters[0]) {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	values := r.Interaction.MessageComponentData().Values
	if len(values) != 1 {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	return h.showSelection(ctx, r.Responder, user, r.Parameters[0], values[0])
}

// validAction limits client-provided route values.
func validAction(s string) bool { return s == "show" || s == "add" || s == "remove" || s == "list" }

// changePage reconstructs a bounded query without server-side state.
func (h Handler) changePage(ctx context.Context, r *discord.InteractionRequest) error {
	user, err := component(r)
	if err != nil {
		return err
	}
	if len(r.Parameters) != 2 || !validAction(r.Parameters[0]) {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	page, err := strconv.Atoi(r.Parameters[1])
	query := ""
	if r.Interaction.Message != nil && len(r.Interaction.Message.Embeds) > 0 && r.Interaction.Message.Embeds[0].Footer != nil {
		query = r.Interaction.Message.Embeds[0].Footer.Text
	}
	if err != nil {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	return h.page(ctx, r.Responder, user, r.Parameters[0], query, page)
}

// add applies a shared card action to the clicking member only.
func (h Handler) add(ctx context.Context, r *discord.InteractionRequest) error {
	user, err := component(r)
	if err != nil {
		return err
	}
	if len(r.Parameters) != 2 {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	return h.showSelection(ctx, r.Responder, user, "add", strings.Join(r.Parameters, ":"))
}

// reviews shows current opinions, hiding withdrawn historical text.
func (h Handler) reviews(ctx context.Context, r *discord.InteractionRequest) error {
	_, err := component(r)
	if err != nil {
		return err
	}
	if len(r.Parameters) != 2 {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	id, err := strconv.ParseUint(r.Parameters[0], 10, 64)
	page, e := strconv.Atoi(r.Parameters[1])
	if err != nil || e != nil || id == 0 || page < 1 || page > 100000 {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	items, err := h.service.Catalog.Reviews(ctx, id)
	if err != nil {
		return err
	}
	lines := []string{}
	start := (page - 1) * reviewPageSize
	if start < 0 || start > len(items) {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	for i := start; i < len(items) && i < start+reviewPageSize; i++ {
		o := items[i]
		text := ""
		if o.Review != nil {
			text = plainOpinion(*o.Review)
		}
		lines = append(lines, fmt.Sprintf("<@%d>: %d/5\n%s", o.UserID, o.Rating, text))
	}
	data := response(ptbr.LetterboxdNoRatings)
	if len(lines) > 0 {
		data.Content = ""
		data.Embeds = []*discordgo.MessageEmbed{{Description: strings.Join(lines, "\n\n")}}
	}
	buttons := []discordgo.MessageComponent{}
	for _, n := range []struct {
		label    string
		page     int
		disabled bool
	}{{ptbr.LetterboxdPrevious, page - 1, page == 1}, {ptbr.LetterboxdNext, page + 1, start+reviewPageSize >= len(items)}} {
		buttons = append(buttons, discordgo.Button{Label: n.label, Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("lb.reviews:%d:%d", id, n.page), Disabled: n.disabled})
	}
	data.Components = []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
	return finish(r.Responder, data)
}

// about presents provider attribution and approved TMDB branding.
func (h Handler) about(_ context.Context, r *command.CommandRequest) error {
	if r.GuildID == "" {
		return command.RejectForbidden(ptbr.LetterboxdGuild)
	}
	data := response(ptbr.LetterboxdCredit + "\nhttps://www.themoviedb.org")
	data.Flags = discordgo.MessageFlagsEphemeral
	data.Embeds = []*discordgo.MessageEmbed{{Image: &discordgo.MessageEmbedImage{URL: "https://www.themoviedb.org/assets/apple-touch-icon-57ed4b3b0450fd5e9a0c20f34e814b82adaa1085c79bdde2f00ca8787b63d2c4.png"}}}
	return r.Responder.Respond(&discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: data})
}
