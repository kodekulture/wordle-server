// Package repository is responsible for the permanent storage of data of this application
package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/kodekulture/wordle-server/game"
	"github.com/kodekulture/wordle-server/game/word"
)

type Player interface {
	// GetByUsername returns a player by username
	GetByUsername(ctx context.Context, username string) (*game.Player, error)

	// UpdatePlayerSession resets the timestamp of the current user session
	UpdatePlayerSession(ctx context.Context, username string, ts int64) error

	// GetByID returns a player by ID
	GetByID(ctx context.Context, id int) (*game.Player, error)

	// Create saves the new player into the database
	Create(ctx context.Context, player game.Player) error
}

type Game interface {
	// GetGames returns all games of a player
	GetGames(ctx context.Context, playerID int) ([]game.Game, error)

	// StartGame saves a game at the beginning of the game
	StartGame(ctx context.Context, g *game.Game) error

	// FinishGame saves a game at the end of the game
	FinishGame(context.Context, *game.Game) error

	// FetchGame returns a game with a given gameID
	FetchGame(context.Context, int, uuid.UUID) (*game.Game, error)
	// WipeGameData is used to delete abandoned games
	WipeGameData(context.Context, uuid.UUID) error
}

type HubBackup interface {
	// Load loads latest hub state
	Load(converter func(g *game.Game) *game.Room) (hub map[uuid.UUID]*game.Room, err error)
	// Dump dump the hub data into a file
	Dump(hub map[uuid.UUID]*game.Room) error
	// Drop deletes the hub data file
	Drop() error
}

type Hub interface {
	CreateGame(context.Context, *game.Game) error
	LoadGame(context.Context, uuid.UUID) (*game.Game, error)
	DeleteGame(context.Context, uuid.UUID) error
	Exists(context.Context, uuid.UUID) bool
	AddGuess(context.Context, uuid.UUID, string, word.Word, bool) error
}
