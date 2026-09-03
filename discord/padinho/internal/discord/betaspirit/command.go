// Package betaspirit exposes Padinho's literal BetaSpirit message command.
package betaspirit

import (
	"context"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

const command = "!betaspirit"

// Register declares the BetaSpirit message command.
func Register(registry *messagecommand.Registry, url string) {
	registry.Command(command, func(_ context.Context, request *messagecommand.Request) error {
		return request.Responder.Reply(url)
	})
}
