// Package voiceactivity records the delivery result of Discord voice activity logs.
package voiceactivity

import "github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"

// Repository persists completed voice activity delivery records.
type Repository interface {
	Create(*entity.VoiceActivityLog) error
}
