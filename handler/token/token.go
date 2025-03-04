// Package token is responsible for generating and validating authentication tokens
package token

import (
	"context"
	"time"

	"github.com/lordvidex/x/auth"

	"github.com/kodekulture/wordle-server/game"
)

//go:generate mockgen -destination=../../internal/mocks/token_handler.go -package=mocks -mock_names Handler=MockTokenHandler -typed . Handler
type Handler interface {
	// Make generates a new token for the given player
	Generate(context.Context, game.Player, time.Duration) (auth.Token, error)
	// Validate validates the given token and returns the player object
	Validate(context.Context, auth.Token) (game.Player, error)
}
