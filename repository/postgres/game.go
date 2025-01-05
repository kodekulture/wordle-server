package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lordvidex/errs/v2"
	"github.com/lordvidex/x/ptr"

	"github.com/kodekulture/wordle-server/game"
	"github.com/kodekulture/wordle-server/game/word"
	"github.com/kodekulture/wordle-server/repository"
	"github.com/kodekulture/wordle-server/repository/postgres/pgen"
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
			BestGuess: pgtype.Text{
				String: s.BestGuess().Word,
				Valid:  s.BestGuess().Word != "",
			},
			BestGuessTime: pgtype.Timestamptz{
				Time:  s.BestGuess().PlayedAt.Time,
				Valid: !s.BestGuess().PlayedAt.Time.IsZero(),
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

func (r *GameRepo) FetchGame(ctx context.Context, playerID int, gameID uuid.UUID) (*game.Game, error) {
	// fetch game
	pgid := pgtype.UUID{Bytes: gameID, Valid: true}
	g, err := r.q.FetchGame(ctx, pgid)
	if err != nil {
		return nil, err
	}
	gm := &game.Game{
		ID:          uuid.UUID(g.ID.Bytes),
		Creator:     g.CreatorUsername,
		CorrectWord: word.New(g.CorrectWord),
		CreatedAt:   g.CreatedAt.Time,
		StartedAt:   toNilTime(g.StartedAt),
		EndedAt:     toNilTime(g.CreatedAt),
	}

	// fetch players
	players, err := r.q.GamePlayers(ctx, pgid)
	if err != nil {
		return nil, err
	}

	// fetch this player
	thisPlayer, err := r.q.GamePlayer(ctx, pgen.GamePlayerParams{
		GameID:   pgid,
		PlayerID: int32(playerID),
	})
	if err != nil {
		return nil, err
	}

	// get session
	setSessions(gm, players, thisPlayer)

	return gm, nil
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
		EndedAt:     toNilTime(g.EndedAt),
		// Sessions: -- sessions are not in the database
	}
}

func setSessions(gm *game.Game, allPlayers []pgen.GamePlayersRow, thisPlayer pgen.GamePlayerRow) {
	rankBoard := game.RankBoard{
		Positions: make(map[string]int, len(allPlayers)),
		Ranks:     make([]*game.Session, len(allPlayers)),
	}
	sessions := make(map[string]*game.Session, len(allPlayers))
	for _, s := range allPlayers {
		var guesses []word.Word
		if s.ID == thisPlayer.ID {
			json.Unmarshal(thisPlayer.PlayedWords, &guesses)
		} else {
			wrd := word.Word{
				Word: s.BestGuess.String,
				PlayedAt: sql.NullTime{
					Time:  s.BestGuessTime.Time,
					Valid: s.BestGuessTime.Valid,
				},
			}
			wrd.Check(gm.CorrectWord)
			guesses = append(guesses, wrd)
		}
		sess := &game.Session{
			Player: game.Player{
				ID:       int(s.ID),
				Username: s.Username,
			},
			Guesses: guesses,
		}
		sess.Resync()
		rankBoard.Ranks[int(s.Rank.Int32)] = sess
		rankBoard.Positions[s.Username] = int(s.Rank.Int32)
		sessions[s.Username] = sess
	}
	gm.Sessions = sessions
	gm.Leaderboard = rankBoard
}
