package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lordvidex/errs/v2"

	"github.com/kodekulture/wordle-server/game"
	"github.com/kodekulture/wordle-server/repository"
	"github.com/kodekulture/wordle-server/repository/postgres/pgen"
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
		Username:  player.Username,
		Password:  player.Password,
		SessionTs: pgtype.Int8{Int64: player.SessionTs, Valid: true},
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
		ID:        int(player.ID),
		Username:  player.Username,
		Password:  player.Password,
		SessionTs: player.SessionTs.Int64,
	}, nil
}

// UpdatePlayerSession ...
func (r *PlayerRepo) UpdatePlayerSession(ctx context.Context, username string, ts int64) error {
	return r.Queries.UpdatePlayerSession(ctx, pgen.UpdatePlayerSessionParams{
		Username:  username,
		SessionTs: pgtype.Int8{Int64: ts, Valid: ts > 0},
	})
}
