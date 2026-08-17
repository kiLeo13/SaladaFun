// Package move contains the /move-all Discord voice-channel interaction.
package move

import "github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"

// Service exposes the Discord voice capabilities used by /move-all.
type Service interface {
	CurrentVoiceChannel(guildID, userID command.Snowflake) (command.Snowflake, bool, error)
	IsVoiceChannel(guildID, channelID command.Snowflake) (bool, error)
	MembersInVoiceChannel(guildID, channelID command.Snowflake) ([]command.Snowflake, error)
	MoveMember(guildID, userID, destinationID command.Snowflake) error
}
