package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/Chat-Map/wordle-server/game/word"
	"github.com/Chat-Map/wordle-server/repository"
	"github.com/Chat-Map/wordle-server/repository/postgres/pgen"
)

var _ repository.Game = new(GameRepo)

type GameRepo struct {
	*pgen.Queries
}

// GetGames implements repository.Game.
func (r *GameRepo) GetGames(ctx context.Context, playerID int) ([]game.Game, error) {
	games, err := r.PlayerGames(ctx, pgen.PlayerGamesParams{
		PlayerID: int32(playerID),
		Limit:    pgtype.Int4{}, // no limit for now
		Offset:   0,             // start from the beginning
	})
	if err != nil {
		return nil, err
	}
	result := make([]game.Game, len(games))
	for i, g := range games {
		result[i] = toGame(g)
	}
	return result, nil
}

func toNilTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	ret := t.Time
	return &ret
}

func NewGameRepo(db pgen.DBTX) *GameRepo {
	return &GameRepo{
		pgen.New(db),
	}
}

func toGame(g pgen.PlayerGamesRow) game.Game {
	return game.Game{
		ID:          uuid.UUID(g.ID.Bytes),
		Creator:     g.CreatorUsername,
		CorrectWord: word.New(g.CorrectWord),
		CreatedAt:   g.CreatedAt.Time,
		StartedAt:   toNilTime(g.StartedAt),
		EndedAt:     toNilTime(g.CreatedAt),
		// Sessions: -- sessions are not in the database
	}

}
