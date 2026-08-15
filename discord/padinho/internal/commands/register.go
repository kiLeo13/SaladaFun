// Package commands is Padinho's single command-composition root. Feature
// packages may expose registration functions and keep handlers together or in
// separate files as each feature warrants.
package commands

import (
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	discordbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/birthday"
)

// Register declares every Padinho command and related interaction route.
func Register(routes *discord.Routes, birthdays discordbirthday.Service) {
	discordbirthday.Register(routes, birthdays)
}
