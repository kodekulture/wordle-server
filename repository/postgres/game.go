package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lordvidex/errs"
	"github.com/lordvidex/x/ptr"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/Chat-Map/wordle-server/game/word"
	"github.com/Chat-Map/wordle-server/repository"
	"github.com/Chat-Map/wordle-server/repository/postgres/pgen"
)

var _ repository.Game = new(GameRepo)

type GameRepo struct {
	db *pgxpool.Pool
	q  *pgen.Queries
}

func (r *GameRepo) StartGame(ctx context.Context, g *game.Game) error {
	if g == nil {
		return errs.B().Msg("game must not be nil in StartGame").Err()
	}
	var (
		tx  pgx.Tx
		err error
	)
	// create a transaction
	tx, err = r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// fetch the game or create it if it doesn't exists
	uid := pgtype.UUID{Bytes: g.ID, Valid: true}

	// Get creator's ID
	player, err := r.q.WithTx(tx).FetchPlayerByUsername(ctx, g.Creator)
	if err != nil {
		return err
	}
	// Create the game
	err = r.q.WithTx(tx).CreateGame(ctx, pgen.CreateGameParams{
		ID:          uid,
		Creator:     player.ID,
		CorrectWord: g.CorrectWord.Word,
		CreatedAt:   pgtype.Timestamptz{Time: g.CreatedAt, Valid: true},
		StartedAt:   pgtype.Timestamptz{Time: ptr.ToObj(g.StartedAt), Valid: g.StartedAt != nil},
	})
	if err != nil {
		return err
	}
	// Create the game player
	args := make([]pgen.CreateGamePlayersParams, 0, len(g.Sessions))
	for _, s := range g.Sessions {
		args = append(args, pgen.CreateGamePlayersParams{
			GameID:   uid,
			PlayerID: int32(s.Player.ID),
		})
	}
	_, err = r.q.WithTx(tx).CreateGamePlayers(ctx, args)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// FinishGame implements repository.Game.
// FinishGame should be mostly an internal function because clients should not save their games themselves.
//
// Update a game should be triggered by the status of the game itself (from the Hub)
func (r *GameRepo) FinishGame(ctx context.Context, g *game.Game) error {
	if g == nil {
		return errs.B().Msg("game must not be nil in StartGame").Err()
	}
	if g.EndedAt == nil {
		return errs.B().Msg("the game has not finished").Err()
	}
	var (
		tx  pgx.Tx
		err error
	)
	// create a transaction
	tx, err = r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	uid := pgtype.UUID{Bytes: g.ID, Valid: true}

	// fetch the game from the database
	gm, err := r.q.WithTx(tx).FetchGame(ctx, uid)
	if err != nil {
		return err
	}
	// Update game, set as finished
	err = r.q.FinishGame(ctx, pgen.FinishGameParams{
		ID:      uid,
		EndedAt: pgtype.Timestamptz{Time: ptr.ToObj(g.EndedAt), Valid: true},
	})
	if err != nil {
		return err
	}
	// Update gamePlayers and set the played words
	for _, s := range g.Sessions {
		r.q.WithTx(tx).UpdateGamePlayer(ctx, pgen.UpdateGamePlayerParams{
			GameID:      gm.ID,
			PlayerID:    int32(s.Player.ID),
			PlayedWords: s.JSON(),
			CorrectGuesses: pgtype.Int4{
				Int32: int32(s.BestGuess().CorrectCount()),
				Valid: len(s.Guesses) > 0,
			},
			CorrectGuessesTime: pgtype.Timestamptz{
				Time:  s.BestGuess().PlayedAt.Time,
				Valid: len(s.Guesses) > 0,
			},
			Finished: pgtype.Timestamptz{
				Time:  s.BestGuess().PlayedAt.Time,
				Valid: s.Won(),
			},
		})
	}
	// commit
	return tx.Commit(ctx)
}

func (r *GameRepo) FetchGame(ctx context.Context, gameID uuid.UUID) (*game.Game, error) {
	g, err := r.q.FetchGame(ctx, pgtype.UUID{Bytes: gameID, Valid: true})
	if err != nil {
		return nil, err
	}
	return &game.Game{
		ID:          uuid.UUID(g.ID.Bytes),
		Creator:     g.CreatorUsername,
		CorrectWord: word.New(g.CorrectWord),
		CreatedAt:   g.CreatedAt.Time,
		StartedAt:   toNilTime(g.StartedAt),
		EndedAt:     toNilTime(g.CreatedAt),
	}, nil
}

// GetGames implements repository.Game.
func (r *GameRepo) GetGames(ctx context.Context, playerID int) ([]game.Game, error) {
	games, err := r.q.PlayerGames(ctx, pgen.PlayerGamesParams{
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

func NewGameRepo(db *pgxpool.Pool) *GameRepo {
	return &GameRepo{
		db: db,
		q:  pgen.New(db),
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
