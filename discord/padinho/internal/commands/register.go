// Package commands is Padinho's single command-composition root. Feature
// packages may expose registration functions and keep handlers together or in
// separate files as each feature warrants.
package commands

import (
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	discordbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/birthday"
	discordmove "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/move"
	discordourochest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/ourochest"
	discordouroquest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/ouroquest"
	discordquote "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/quote"
)

// Gateway exposes the Discord capabilities shared by command features.
type Gateway interface {
	discordbirthday.GuildLookup
	discordmove.Service
}

// Register declares every Padinho command and related interaction route.
func Register(
	routes *discord.Routes,
	birthdays discordbirthday.Service,
	quotes discordquote.Service,
	gateway Gateway,
	ouroChest *discordourochest.Listener,
	ouroQuest *discordouroquest.Listener,
) {
	discordbirthday.Register(routes, birthdays, gateway)
	discordmove.Register(routes, gateway)
	discordquote.Register(routes.Messages(), quotes)
	discordourochest.Register(routes.Messages(), ouroChest)
	discordouroquest.Register(routes.Messages(), ouroQuest)
}
