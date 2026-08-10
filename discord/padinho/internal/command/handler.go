package command

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// HandlerFunc executes a command. Functions and methods with this signature can
// be registered directly.
type HandlerFunc func(context.Context, *CommandRequest) error

// Middleware wraps a command handler. A middleware may call next or stop the
// chain by returning an error or Rejection.
type Middleware func(HandlerFunc) HandlerFunc

// RejectionCode identifies an expected command rejection.
type RejectionCode string

const (
	RejectionForbidden RejectionCode = "forbidden"
	RejectionCooldown  RejectionCode = "cooldown"
)

// Rejection is an expected, user-facing refusal to execute a command.
type Rejection struct {
	Code       RejectionCode
	Message    string
	RetryAfter time.Duration
}

func (r *Rejection) Error() string {
	if r.Message != "" {
		return r.Message
	}
	return fmt.Sprintf("command rejected: %s", r.Code)
}

// RejectForbidden rejects a request with a user-facing reason.
func RejectForbidden(reason string) error {
	return &Rejection{Code: RejectionForbidden, Message: reason}
}

// RejectCooldown rejects a request until its cooldown elapses.
func RejectCooldown(retryAfter time.Duration) error {
	return &Rejection{
		Code:       RejectionCooldown,
		Message:    "command is on cooldown",
		RetryAfter: retryAfter,
	}
}

// AsRejection extracts an expected command rejection.
func AsRejection(err error) (*Rejection, bool) {
	var rejection *Rejection
	if !errors.As(err, &rejection) {
		return nil, false
	}
	return rejection, true
}

func compose(handler HandlerFunc, middlewares ...Middleware) HandlerFunc {
	for index := len(middlewares) - 1; index >= 0; index-- {
		handler = middlewares[index](handler)
	}
	return handler
}
