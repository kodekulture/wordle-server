// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: game.sql

package pgen

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createGame = `-- name: CreateGame :exec
INSERT INTO game (id, creator, correct_word, created_at, started_at) VALUES ($1, $2, $3, $4, $5)
`

type CreateGameParams struct {
	ID          pgtype.UUID
	Creator     int32
	CorrectWord string
	CreatedAt   pgtype.Timestamptz
	StartedAt   pgtype.Timestamptz
}

func (q *Queries) CreateGame(ctx context.Context, arg CreateGameParams) error {
	_, err := q.db.Exec(ctx, createGame,
		arg.ID,
		arg.Creator,
		arg.CorrectWord,
		arg.CreatedAt,
		arg.StartedAt,
	)
	return err
}

type CreateGamePlayersParams struct {
	GameID   pgtype.UUID
	PlayerID int32
}

const deleteGame = `-- name: DeleteGame :exec
DELETE FROM game WHERE id = $1
`

func (q *Queries) DeleteGame(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteGame, id)
	return err
}

const deleteGamePlayers = `-- name: DeleteGamePlayers :exec
DELETE FROM game_player WHERE game_id = $1
`

func (q *Queries) DeleteGamePlayers(ctx context.Context, gameID pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteGamePlayers, gameID)
	return err
}

const fetchGame = `-- name: FetchGame :one
SELECT p.username AS creator_username, g.id, g.creator, g.correct_word, g.created_at, g.started_at, g.ended_at from game g
JOIN player p ON g.creator = p.id WHERE g.id = $1
`

type FetchGameRow struct {
	CreatorUsername string
	ID              pgtype.UUID
	Creator         int32
	CorrectWord     string
	CreatedAt       pgtype.Timestamptz
	StartedAt       pgtype.Timestamptz
	EndedAt         pgtype.Timestamptz
}

func (q *Queries) FetchGame(ctx context.Context, id pgtype.UUID) (FetchGameRow, error) {
	row := q.db.QueryRow(ctx, fetchGame, id)
	var i FetchGameRow
	err := row.Scan(
		&i.CreatorUsername,
		&i.ID,
		&i.Creator,
		&i.CorrectWord,
		&i.CreatedAt,
		&i.StartedAt,
		&i.EndedAt,
	)
	return i, err
}

const finishGame = `-- name: FinishGame :exec
UPDATE game SET ended_at = coalesce($2, NOW()) WHERE id = $1
`

type FinishGameParams struct {
	ID      pgtype.UUID
	EndedAt pgtype.Timestamptz
}

func (q *Queries) FinishGame(ctx context.Context, arg FinishGameParams) error {
	_, err := q.db.Exec(ctx, finishGame, arg.ID, arg.EndedAt)
	return err
}

const gamePlayer = `-- name: GamePlayer :one
SELECT p.id, p.username, gp.game_id, gp.player_id, gp.played_words, gp.best_guess, gp.best_guess_time, gp.finished, gp.rank FROM game_player gp
JOIN player p ON gp.player_id = p.id
WHERE gp.game_id = $1 AND gp.player_id = $2
`

type GamePlayerParams struct {
	GameID   pgtype.UUID
	PlayerID int32
}

type GamePlayerRow struct {
	ID            int32
	Username      string
	GameID        pgtype.UUID
	PlayerID      int32
	PlayedWords   []byte
	BestGuess     pgtype.Text
	BestGuessTime pgtype.Timestamptz
	Finished      pgtype.Timestamptz
	Rank          pgtype.Int4
}

// returns the full data of a player in a game
func (q *Queries) GamePlayer(ctx context.Context, arg GamePlayerParams) (GamePlayerRow, error) {
	row := q.db.QueryRow(ctx, gamePlayer, arg.GameID, arg.PlayerID)
	var i GamePlayerRow
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.GameID,
		&i.PlayerID,
		&i.PlayedWords,
		&i.BestGuess,
		&i.BestGuessTime,
		&i.Finished,
		&i.Rank,
	)
	return i, err
}

const gamePlayers = `-- name: GamePlayers :many
SELECT p.id, p.username, gp.best_guess, gp.best_guess_time, gp.finished, gp.rank, jsonb_array_length(gp.played_words)::int as total_words
FROM game_player gp 
JOIN player p ON gp.player_id = p.id 
WHERE gp.game_id = $1
`

type GamePlayersRow struct {
	ID            int32
	Username      string
	BestGuess     pgtype.Text
	BestGuessTime pgtype.Timestamptz
	Finished      pgtype.Timestamptz
	Rank          pgtype.Int4
	TotalWords    int32
}

// returns all the players that played this game but only returns their best word
func (q *Queries) GamePlayers(ctx context.Context, gameID pgtype.UUID) ([]GamePlayersRow, error) {
	rows, err := q.db.Query(ctx, gamePlayers, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GamePlayersRow
	for rows.Next() {
		var i GamePlayersRow
		if err := rows.Scan(
			&i.ID,
			&i.Username,
			&i.BestGuess,
			&i.BestGuessTime,
			&i.Finished,
			&i.Rank,
			&i.TotalWords,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const playerGames = `-- name: PlayerGames :many
SELECT g.id, g.correct_word, g.created_at, g.started_at, g.ended_at, 
  p.id AS creator_id, p.username AS creator_username,
  gp.player_id, gp.played_words, gp.best_guess, gp.best_guess_time, gp.finished, gp.rank
FROM game g
JOIN game_player gp ON g.id = gp.game_id
JOIN player p ON g.creator = p.id
WHERE gp.player_id = $1
ORDER BY gp.finished DESC
LIMIT $3 OFFSET $2
`

type PlayerGamesParams struct {
	PlayerID int32
	Offset   int32
	Limit    pgtype.Int4
}

type PlayerGamesRow struct {
	ID              pgtype.UUID
	CorrectWord     string
	CreatedAt       pgtype.Timestamptz
	StartedAt       pgtype.Timestamptz
	EndedAt         pgtype.Timestamptz
	CreatorID       int32
	CreatorUsername string
	PlayerID        int32
	PlayedWords     []byte
	BestGuess       pgtype.Text
	BestGuessTime   pgtype.Timestamptz
	Finished        pgtype.Timestamptz
	Rank            pgtype.Int4
}

func (q *Queries) PlayerGames(ctx context.Context, arg PlayerGamesParams) ([]PlayerGamesRow, error) {
	rows, err := q.db.Query(ctx, playerGames, arg.PlayerID, arg.Offset, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PlayerGamesRow
	for rows.Next() {
		var i PlayerGamesRow
		if err := rows.Scan(
			&i.ID,
			&i.CorrectWord,
			&i.CreatedAt,
			&i.StartedAt,
			&i.EndedAt,
			&i.CreatorID,
			&i.CreatorUsername,
			&i.PlayerID,
			&i.PlayedWords,
			&i.BestGuess,
			&i.BestGuessTime,
			&i.Finished,
			&i.Rank,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateGamePlayer = `-- name: UpdateGamePlayer :exec
UPDATE game_player SET played_words=$3, best_guess=$4, best_guess_time=$5, finished=$6, rank=$7 
WHERE game_id=$1 AND player_id=$2
`

type UpdateGamePlayerParams struct {
	GameID        pgtype.UUID
	PlayerID      int32
	PlayedWords   []byte
	BestGuess     pgtype.Text
	BestGuessTime pgtype.Timestamptz
	Finished      pgtype.Timestamptz
	Rank          pgtype.Int4
}

// This updates the player stats at the end of the game
func (q *Queries) UpdateGamePlayer(ctx context.Context, arg UpdateGamePlayerParams) error {
	_, err := q.db.Exec(ctx, updateGamePlayer,
		arg.GameID,
		arg.PlayerID,
		arg.PlayedWords,
		arg.BestGuess,
		arg.BestGuessTime,
		arg.Finished,
		arg.Rank,
	)
	return err
}
