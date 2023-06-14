// Package repository is responsible for the permanent storage of data of this application
package repository

import (
	"context"

	"github.com/Chat-Map/wordle-server/game"
)

type Player interface {
	// GetByUsername returns a player by username
	GetByUsername(ctx context.Context, username string) (*game.Player, error)

	// GetByID returns a player by ID
	GetByID(ctx context.Context, id int) (*game.Player, error)

	// Create saves the new player into the database
	Create(ctx context.Context, player game.Player) error
}

type Game interface {
	// GetGames returns all games of a player
	GetGames(ctx context.Context, playerID int) ([]game.Game, error)

	// SaveGame saves a game into the dababase as well as the player's sessions
	SaveGame(ctx context.Context, g *game.Game) error
}
