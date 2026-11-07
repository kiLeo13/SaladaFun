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
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/enus"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const pageSize = app.SearchPageSize
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
	sortOption := command.StringOption("sort", ptbr.LetterboxdSort).Choices(command.OptionChoice{Name: enus.LetterboxdSortDate, Value: "date"}, command.OptionChoice{Name: enus.LetterboxdSortRating, Value: "rating"})
	orderOption := command.StringOption("order", ptbr.LetterboxdOrder).Choices(command.OptionChoice{Name: enus.LetterboxdOrderAscending, Value: "asc"}, command.OptionChoice{Name: enus.LetterboxdOrderDescending, Value: "desc"})
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
	routes.Component("lb.reviews-page", h.reviewsPage)
	routes.Component("lb.watch", h.openWatch)
	routes.Component("lb.edit", h.openWatch)
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
				for _, t := range items.Items {
					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: Ellipsis(label(t), 100), Value: app.Identity(t.MediaType, t.TMDBID)})
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
	if t.ReleaseYear != nil {
		year = strconv.Itoa(*t.ReleaseYear)
	}
	return fmt.Sprintf("%s (%s, %s)", app.DisplayTitle(&t), year, kind)
}

// Ellipsis limits Discord text to n UTF-16 units, including a trailing "..."
// when truncated. It preserves whole Unicode characters and uses fewer dots
// when n is less than three; nonpositive limits return an empty string.
func Ellipsis(s string, n int) string {
	if n <= 0 {
		return ""
	}
	suffix := "..."
	if n < len(suffix) {
		suffix = suffix[:n]
	}
	units := 0
	cut := len(s)
	for i, r := range s {
		width := 1
		if r > 0xffff {
			width = 2
		}
		units += width
		if units > n-len(suffix) && cut == len(s) {
			cut = i
		}
		if units > n {
			return s[:cut] + suffix
		}
	}
	return s
}

// page lists remote matches or locally stored interests with reconstructible navigation.
func (h Handler) page(ctx context.Context, r command.Responder, user uint64, action, query string, page int) error {
	if utf8.RuneCountInString(query) > 200 || page < 1 || page > 500 {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	results, cached, err := h.service.Search(ctx, user, action, query, page)
	items := results.Items
	if err != nil {
		return err
	}
	if action == "show" && !cached && results.TotalResults == 1 && len(items) == 1 {
		return h.showSelection(ctx, r, user, action, app.Identity(items[0].MediaType, items[0].TMDBID))
	}
	more := page < results.TotalPages
	if action == "remove" || action == "list" {
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
	return finish(r, resultPage(action, query, page, items, cached, more, results.TotalResults))
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

// component validates actor context and acknowledges privately unless explicitly public.
func component(r *discord.InteractionRequest, private ...bool) (uint64, error) {
	if err := deferReply(r.Responder, r.GuildID, private...); err != nil {
		return 0, err
	}
	id, err := strconv.ParseUint(string(r.Actor.UserID), 10, 64)
	if err != nil || id == 0 {
		return 0, command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	return id, nil
}

// updateComponent acknowledges pagination by deferring an edit of its source message.
func updateComponent(r *discord.InteractionRequest) (uint64, error) {
	if r.GuildID == "" {
		return 0, command.RejectForbidden(ptbr.LetterboxdGuild)
	}
	id, err := strconv.ParseUint(string(r.Actor.UserID), 10, 64)
	if err != nil || id == 0 {
		return 0, command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	err = r.Responder.Respond(&discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
	return id, err
}

// selectTitle resolves one selected remote identity.
func (h Handler) selectTitle(ctx context.Context, r *discord.InteractionRequest) error {
	private := len(r.Parameters) != 1 || r.Parameters[0] != "show"
	user, err := component(r, private)
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
	user, err := updateComponent(r)
	if err != nil {
		return err
	}
	if len(r.Parameters) != 2 || !validAction(r.Parameters[0]) {
		return command.RejectForbidden(ptbr.LetterboxdInvalid)
	}
	page, err := strconv.Atoi(r.Parameters[1])
	query := ""
	if r.Interaction.Message != nil && len(r.Interaction.Message.Embeds) > 0 && r.Interaction.Message.Embeds[0].Footer != nil {
		embed := r.Interaction.Message.Embeds[0]
		query = embed.Footer.Text
		if strings.HasPrefix(embed.Title, ptbr.LetterboxdResultsPrefix) || strings.HasPrefix(embed.Title, ptbr.LetterboxdCachedResultsPrefix) {
			query = strings.TrimPrefix(query, ptbr.LetterboxdQueryPrefix)
		}
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

// reviews opens opinions and recognizes pagination buttons on already-rendered legacy review pages.
func (h Handler) reviews(ctx context.Context, r *discord.InteractionRequest) error {
	update := false
	if len(r.Parameters) == 2 {
		page, err := strconv.Atoi(r.Parameters[1])
		update = err == nil && page > 1
	}
	if r.Interaction != nil && r.Interaction.Message != nil && len(r.Interaction.Message.Embeds) > 0 {
		embed := r.Interaction.Message.Embeds[0]
		update = update || (embed.Footer != nil && strings.HasPrefix(embed.Title, strings.TrimSuffix(ptbr.LetterboxdReviewTitle, "%s")) && strings.HasPrefix(embed.Footer.Text, strings.SplitN(ptbr.LetterboxdReviewTotal, "%d", 2)[0]))
	}
	return h.reviewResponse(ctx, r, update)
}

// reviewsPage handles the dedicated restart-safe pagination route.
func (h Handler) reviewsPage(ctx context.Context, r *discord.InteractionRequest) error {
	return h.reviewResponse(ctx, r, true)
}

// reviewResponse renders current opinions after acknowledging a new reply or a source-message update.
func (h Handler) reviewResponse(ctx context.Context, r *discord.InteractionRequest, update bool) error {
	var err error
	if update {
		_, err = updateComponent(r)
	} else {
		_, err = component(r, false)
	}
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
	title, err := h.service.Catalog.TitleByID(ctx, id)
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
		text := ptbr.LetterboxdReviewAbsent
		if o.Review != nil {
			if review := plainOpinion(*o.Review); review != "" {
				text = review
			}
		}
		lines = append(lines, fmt.Sprintf(ptbr.LetterboxdReviewEntry, o.UserID, o.Rating, text))
	}
	description := strings.Join(lines, "\n\n")
	if len(lines) == 0 {
		description = ptbr.LetterboxdNoRatings
	}
	total := fmt.Sprintf(ptbr.LetterboxdReviewTotal, len(items))
	if len(items) == 1 {
		total = ptbr.LetterboxdReviewTotalSingle
	}
	embed := &discordgo.MessageEmbed{
		Title:       Ellipsis(fmt.Sprintf(ptbr.LetterboxdReviewTitle, plainOpinion(app.DisplayTitle(title))), 256),
		Description: description,
		Footer:      &discordgo.MessageEmbedFooter{Text: total},
	}
	data := response("")
	data.Embeds = []*discordgo.MessageEmbed{embed}
	buttons := []discordgo.MessageComponent{}
	for _, n := range []struct {
		label    string
		page     int
		disabled bool
	}{{ptbr.LetterboxdPrevious, page - 1, page == 1}, {ptbr.LetterboxdNext, page + 1, start+reviewPageSize >= len(items) || page >= 100000}} {
		buttons = append(buttons, discordgo.Button{Label: n.label, Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("lb.reviews-page:%d:%d", id, n.page), Disabled: n.disabled})
	}
	data.Components = []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
	return finish(r.Responder, data)
}

// about presents localized feature guidance and TMDB attribution.
func (h Handler) about(_ context.Context, r *command.CommandRequest) error {
	if r.GuildID == "" {
		return command.RejectForbidden(ptbr.LetterboxdGuild)
	}
	return r.Responder.Respond(&discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: aboutPage()})
}
