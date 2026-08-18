package move

import (
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const commandName = "move-all"

// Register declares the /move-all command.
func Register(routes *discord.Routes, service Service) {
	handler := Handler{service: service}
	routes.Commands().Slash(
		commandName,
		ptbr.MoveAllCommandDescription,
		handler.MoveAll,
		command.ChannelOption("destination", ptbr.MoveAllDestinationDescription).VoiceOnly().Required(),
		command.ChannelOption("origin", ptbr.MoveAllOriginDescription).VoiceOnly(),
	)
}
