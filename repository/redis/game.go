package redis

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kodekulture/wordle-server/game"
	"github.com/kodekulture/wordle-server/game/word"
	redis9 "github.com/redis/go-redis/v9"
)

const (
	GameExp = time.Hour
)

var (
	ErrNoGame = errors.New("game does not exist")
)

type Game struct {
	Game    *game.Game
	Players []string `json:"p"`
}

// MarshalJSON returns JSON value of Game without Sessions and Leaderboards as they already exist in redis.
func (g Game) MarshalJSON() ([]byte, error) {
	internal := *g.Game
	internal.Sessions = nil

	m := map[string]any{
		"p":    g.Players,
		"Game": &internal,
	}
	return json.Marshal(m)
}

// GameRepository ...
type GameRepository struct {
	cl *redis9.Client
}

// CreateGame ...
func (r GameRepository) CreateGame(ctx context.Context, g *game.Game) error {
	if g == nil {
		return errors.New("nil game")
	}
	// check if game already exists
	if r.Exists(ctx, g.ID) {
		return errors.New("game already exists")
	}

	// create game metadata
	players := g.Players()
	rg := Game{Game: g, Players: players}
	b, err := json.Marshal(rg)
	if err != nil {
		return err
	}

	return r.cl.SetEx(ctx, gm(g.ID), string(b), GameExp).Err()
}

// GetGame returns only game metadata without player's sessions
func (r GameRepository) GetGame(ctx context.Context, gameID uuid.UUID) (*game.Game, error) {
	g, err := r.getGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	return g.Game, nil
}

func (r GameRepository) getGame(ctx context.Context, gameID uuid.UUID) (Game, error) {
	str, err := r.cl.Get(ctx, gm(gameID)).Result()
	if err != nil {
		return Game{}, err
	}

	var g Game
	if err = json.Unmarshal([]byte(str), &g); err != nil {
		return Game{}, err
	}
	return g, nil
}

// DeleteGame ...
func (r GameRepository) DeleteGame(ctx context.Context, gameID uuid.UUID) error {
	rg, err := r.getGame(ctx, gameID)
	if err != nil {
		// game does not exist, delete is no-op
		return nil
	}

	keys := []string{gm(gameID), keyed(gm(gameID), "leaderboard")}
	for _, p := range rg.Players {
		keys = append(keys, keyed(gm(gameID), ss(p)))
	}
	return r.cl.Del(ctx, keys...).Err()
}

// LoadGame loads full game data and resynced player sessions from storage
func (r GameRepository) LoadGame(ctx context.Context, gameID uuid.UUID) (*game.Game, error) {
	rg, err := r.getGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	// load player sessions
	sess, err := r.getSessions(ctx, gameID, rg.Players)
	if err != nil {
		return nil, err
	}

	ldb := game.NewRankBoard(sess)

	g := rg.Game
	g.Leaderboard = ldb
	g.Sessions = sess
	g.Resync()

	return g, nil
}

// getSessions returns resynced player's game sessions.
func (r GameRepository) getSessions(ctx context.Context, gameID uuid.UUID, players []string) (map[string]*game.Session, error) {
	sess := make(map[string]*game.Session)

	pipe := r.cl.Pipeline()
	for _, p := range players {
		pipe.LRange(ctx, keyed(gm(gameID), ss(p)), 0, -1)
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	for i, cmd := range cmds {
		player := players[i]
		s := &game.Session{
			Player: game.Player{
				Username: player,
			},
		}
		val, err := cmd.(*redis9.StringSliceCmd).Result()
		if err != nil {
			if errors.Is(err, redis9.Nil) {
				sess[player] = s
				continue
			}
			return nil, err
		}

		for _, w := range val {
			var guess word.Word
			err = json.Unmarshal([]byte(w), &guess)
			if err != nil {
				return nil, err
			}
			s.Guesses = append(s.Guesses, guess)
		}
		sess[player] = s
	}

	return sess, nil
}

func (r GameRepository) Exists(ctx context.Context, gameID uuid.UUID) bool {
	res := r.cl.Exists(ctx, gm(gameID)).Val()
	return res == 1
}

func (r GameRepository) AddGuess(ctx context.Context, gameID uuid.UUID, player string, guess word.Word, isBest bool) error {
	if !r.Exists(ctx, gameID) {
		return ErrNoGame
	}

	_, err := r.cl.Pipelined(ctx, func(pipe redis9.Pipeliner) error {
		err := pipe.RPush(ctx, keyed(gm(gameID), ss(player)), guess).Err()
		if err != nil {
			return err
		}
		err = pipe.ExpireLT(ctx, keyed(gm(gameID), ss(player)), GameExp).Err()
		if err != nil {
			return err
		}

		if isBest {
			err = pipe.HSet(ctx, keyed(gm(gameID), "leaderboard"), player, guess).Err()
			if err != nil {
				return err
			}
			err = pipe.ExpireLT(ctx, keyed(gm(gameID), "leaderboard"), GameExp).Err()
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (r GameRepository) GetGuesses(ctx context.Context, gameID uuid.UUID, player string) ([]word.Word, error) {
	if !r.Exists(ctx, gameID) {
		return nil, ErrNoGame
	}

	res, err := r.cl.LRange(ctx, keyed(gm(gameID), ss(player)), 0, -1).Result()
	if err != nil {
		return nil, err
	}

	var guesses []word.Word
	for _, g := range res {
		var guess word.Word
		err := json.Unmarshal([]byte(g), &guess)
		if err != nil {
			return nil, err
		}
		guesses = append(guesses, guess)
	}
	return guesses, nil
}

func (r GameRepository) GetBestGuess(ctx context.Context, gameID uuid.UUID, player string) (*word.Word, error) {
	if !r.Exists(ctx, gameID) {
		return nil, ErrNoGame
	}

	res, err := r.cl.HGet(ctx, keyed(gm(gameID), "leaderboard"), player).Result()
	if err != nil {
		return nil, err
	}
	var w word.Word
	if err = json.Unmarshal([]byte(res), &w); err != nil {
		return nil, err
	}

	return &w, nil
}

func keyed(s ...string) string {
	return strings.Join(s, ":")
}

// gm returns game:<gid>
func gm(gameID uuid.UUID) string {
	return keyed("game", gameID.String())
}

// ss returns session:<player>
func ss(player string) string {
	return keyed("session", player)
}
