package birthday

import (
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const (
	commandName      = "birthdays"
	pageRoute        = "birthdays.page"
	addBirthdayRoute = "birthdays.add"
	inspectRoute     = "birthdays.inspect"
	editRoute        = "birthdays.edit"
	editSelectRoute  = "birthdays.edit-select"
	editFieldRoute   = "birthdays.edit-field"
	editSubmitRoute  = "birthdays.edit-submit"
)

// Register declares every slash, component, and modal route for birthdays.
func Register(routes *discord.Routes, service Service) {
	handler := Handler{service: service}
	routes.Commands().Slash(commandName, ptbr.BirthdayCommandDescription, handler.List, monthOption())
	routes.Component(pageRoute, handler.ChangePage)
	routes.Component(addBirthdayRoute, handler.OpenModal)
	routes.Component(inspectRoute, handler.Inspect)
	routes.Component(editRoute, handler.OpenDashboard)
	routes.Component(editSelectRoute, handler.SelectDashboardUser)
	routes.Component(editFieldRoute, handler.OpenEditModal)
	routes.Modal(addBirthdayRoute, handler.Submit)
	routes.Modal(editSubmitRoute, handler.SubmitEdit)
}
