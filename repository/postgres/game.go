package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lordvidex/errs"
	"github.com/lordvidex/x/ptr"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/Chat-Map/wordle-server/game/word"
	"github.com/Chat-Map/wordle-server/repository"
	"github.com/Chat-Map/wordle-server/repository/postgres/pgen"
)

var _ repository.Game = new(GameRepo)

type GameRepo struct {
	db DBTX
	*pgen.Queries
}

// SaveGame implements repository.Game.
// SaveGame should be mostly an internal function because clients should not save their games themselves.
//
// Saving a game should be triggered by the status of the game itself (from the Hub)
func (r *GameRepo) SaveGame(ctx context.Context, g *game.Game) error {
	if g == nil {
		return errs.B().Msg("game must not be nil in SaveGame").Err()
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
	_, err = r.WithTx(tx).FetchGame(ctx, uid)
	if err != nil {
		if err != pgx.ErrNoRows {
			return err
		}
		// Create the game if it does not exist
		err = r.createGame(ctx, tx, g)
		if err != nil {
			return err
		}
	}

	// fetch all of the players in the game
	p, err := r.WithTx(tx).GamePlayers(ctx, uid)
	if err != nil {
		return err
	}
	var playerMap = make(map[string]pgen.GamePlayersRow)
	for _, player := range p {
		playerMap[player.Username] = player
	}
	// update the game
	switch {
	case g.EndedAt != nil:
		err = r.WithTx(tx).FinishGame(ctx, pgen.FinishGameParams{
			ID: uid,
			EndedAt: pgtype.Timestamptz{
				Time:  ptr.ToObj(g.EndedAt),
				Valid: g.EndedAt != nil,
			}})
	case g.StartedAt != nil:
		err = r.WithTx(tx).StartGame(ctx, pgen.StartGameParams{
			ID: uid,
			StartedAt: pgtype.Timestamptz{
				Time:  ptr.ToObj(g.StartedAt),
				Valid: g.StartedAt != nil,
			}})
	default:
		return errs.B().Msg("game can only be saved when starting or ending").Err()
	}
	if err != nil {
		return err
	}
	// update the player's sessions
	var errors []error
	for username, session := range g.Sessions {
		player, ok := playerMap[username]
		if !ok {
			errors = append(errors, errs.B().Msgf("player %s not found in game %s and data was not updated", username, g.ID).Err())
			continue
		}
		err = r.WithTx(tx).UpdatePlayerStats(ctx, pgen.UpdatePlayerStatsParams{
			GameID:   uid,
			PlayerID: player.ID,
			Finished: func() pgtype.Timestamptz {
				if session.Ended() {
					return pgtype.Timestamptz{}
				}
				t := session.Guesses[len(session.Guesses)-1].PlayedAt
				return pgtype.Timestamptz{
					Time:  t.Time,
					Valid: t.Valid,
				}
			}(),
			PlayedWords: func() []byte {
				b, err := json.Marshal(session.Guesses)
				if err != nil {
					errors = append(errors, errs.B().Msgf("failed to convert played words to json for player %d", player.ID).
						Err())
					return nil
				}
				return b
			}(),
			// TODO: fill these
			CorrectGuesses:     pgtype.Int4{},
			CorrectGuessesTime: pgtype.Timestamptz{},
		})
		if err != nil {
			errors = append(errors, err)
		}
	}
	// commit
	err = tx.Commit(ctx)
	if err != nil {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return joinErrs(errors...)
	}
	return nil
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

func (r *GameRepo) createGame(ctx context.Context, tx pgx.Tx, g *game.Game) error {
	// Get creator's ID
	player, err := r.WithTx(tx).FetchPlayerByUsername(ctx, g.Creator)
	if err != nil {
		return err
	}
	// Create the game
	err = r.WithTx(tx).CreateGame(ctx, pgen.CreateGameParams{
		ID:          pgtype.UUID{Bytes: g.ID, Valid: true},
		Creator:     player.ID,
		CorrectWord: g.CorrectWord.Word,
	})
	if err != nil {
		return err
	}
	return nil
}

func toNilTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	ret := t.Time
	return &ret
}

func joinErrs(errs ...error) error {
	var b strings.Builder
	for _, err := range errs {
		b.WriteString(err.Error())
		b.WriteString("\n\n")
	}
	return errors.New(b.String())
}

type DBTX interface {
	pgen.DBTX
	pgx.Tx
}

func NewGameRepo(db DBTX) *GameRepo {
	return &GameRepo{
		db:      db,
		Queries: pgen.New(db),
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
