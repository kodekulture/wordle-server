// Package token is responsible for generating and validating authentication tokens
package token

import (
	"context"

	"github.com/lordvidex/x/auth"

	"github.com/Chat-Map/wordle-server/game"
)

type Handler interface {
	// Make generates a new token for the given player
	Generate(context.Context, game.Player) (auth.Token, error)
	// Validate validates the given token and returns the player object
	Validate(context.Context, auth.Token) (game.Player, error)
}
