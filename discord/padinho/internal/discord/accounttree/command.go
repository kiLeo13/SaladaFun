// Package accounttree exposes the literal Discord account-hierarchy command.
package accounttree

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	appaccounttree "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/accounttree"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

const childrenTreeCommand = "!childrentree"

// Service loads a complete rooted Discord account hierarchy.
type Service interface {
	Tree(userID uint64) (*appaccounttree.Tree, error)
}

// MemberLookup resolves a guild member's display name from its Discord ID.
type MemberLookup interface {
	GuildMemberDisplayName(guildID, userID command.Snowflake) (string, bool, error)
}

// componentResponder sends a Components V2 reply to one message command.
type componentResponder interface {
	ReplyWithComponents([]discordgo.MessageComponent) error
}

// Register declares the account-tree message command.
func Register(registry *messagecommand.Registry, service Service, members MemberLookup) {
	registry.Command(childrenTreeCommand, func(ctx context.Context, request *messagecommand.Request) error {
		return handle(ctx, request, service, members)
	})
}

// handle renders the complete account hierarchy that contains the requested user.
func handle(_ context.Context, request *messagecommand.Request, service Service, members MemberLookup) error {
	targetID, err := requestedUserID(request)
	if err != nil {
		return request.Responder.Reply(err.Error())
	}
	if _, found, err := members.GuildMemberDisplayName(request.GuildID, targetID); err != nil {
		return err
	} else if !found {
		return request.Responder.Reply(ptbr.AccountTreeUserNotFound)
	}
	parsedTargetID, err := strconv.ParseUint(string(targetID), 10, 64)
	if err != nil || parsedTargetID == 0 {
		return request.Responder.Reply(ptbr.AccountTreeInvalidID)
	}
	tree, err := service.Tree(parsedTargetID)
	if err != nil {
		return err
	}
	names, err := lookupNames(request.GuildID, tree.Root, members)
	if err != nil {
		return err
	}
	responder, ok := request.Responder.(componentResponder)
	if !ok {
		return errors.New("message command responder does not support Components V2")
	}
	return responder.ReplyWithComponents(render(tree, names))
}

// requestedUserID returns the invoking user or the first explicit entity argument.
func requestedUserID(request *messagecommand.Request) (command.Snowflake, error) {
	if len(request.Arguments) == 0 {
		if request.Actor.UserID == "" {
			return "", errors.New(ptbr.AccountTreeInvalidID)
		}
		return request.Actor.UserID, nil
	}
	userID, valid := messagecommand.ExtractSnowflake(request.Arguments[0])
	if !valid {
		return "", errors.New(ptbr.AccountTreeInvalidID)
	}
	return userID, nil
}

// lookupNames resolves every displayed account while retaining IDs for stale descendants.
func lookupNames(guildID command.Snowflake, node *appaccounttree.Node, members MemberLookup) (map[uint64]string, error) {
	names := make(map[uint64]string)
	var visit func(*appaccounttree.Node) error
	visit = func(current *appaccounttree.Node) error {
		name, found, err := members.GuildMemberDisplayName(guildID, command.Snowflake(strconv.FormatUint(current.UserID, 10)))
		if err != nil {
			return err
		}
		if !found {
			name = strconv.FormatUint(current.UserID, 10)
		}
		names[current.UserID] = name
		for _, child := range current.Children {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(node); err != nil {
		return nil, fmt.Errorf("resolve account tree member names: %w", err)
	}
	return names, nil
}
