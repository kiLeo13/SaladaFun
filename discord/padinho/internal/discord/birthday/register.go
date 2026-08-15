package birthday

import (
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const (
	commandName      = "birthdays"
	pageRoute        = "birthdays.page"
	addBirthdayRoute = "birthdays.add"
)

// Register declares every slash, component, and modal route for birthdays.
func Register(routes *discord.Routes, service Service) {
	handler := Handler{service: service}
	routes.Commands().Slash(commandName, ptbr.BirthdayCommandDescription, handler.List)
	routes.Component(pageRoute, handler.ChangePage)
	routes.Component(addBirthdayRoute, handler.OpenModal)
	routes.Modal(addBirthdayRoute, handler.Submit)
}
