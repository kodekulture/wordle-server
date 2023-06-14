package postgres

import (
	"context"

	"github.com/lordvidex/errs"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/Chat-Map/wordle-server/repository"
	"github.com/Chat-Map/wordle-server/repository/postgres/pgen"
)

var _ repository.Player = new(PlayerRepo)

type PlayerRepo struct {
	*pgen.Queries
}

func NewPlayerRepo(db pgen.DBTX) *PlayerRepo {
	return &PlayerRepo{
		pgen.New(db),
	}
}

// Create implements repository.Player.
func (r *PlayerRepo) Create(ctx context.Context, player game.Player) error {
	return r.AddPlayer(ctx, pgen.AddPlayerParams{
		Username: player.Username,
		Password: player.Password,
	})
}

// GetByID implements repository.Player.
func (r *PlayerRepo) GetByID(ctx context.Context, id int) (*game.Player, error) {
	return nil, errs.B().Code(errs.Internal).Msg("not implemented").Err()
}

// GetByUsername implements repository.Player.
func (r *PlayerRepo) GetByUsername(ctx context.Context, username string) (*game.Player, error) {
	player, err := r.FetchPlayerByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return &game.Player{
		ID:       int(player.ID),
		Username: player.Username,
		Password: player.Password,
	}, nil
}
