// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.18.0
// source: game.sql

package pgen

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const gamePlayers = `-- name: GamePlayers :many
SELECT p.id, p.username, gp.correct_guesses, gp.correct_guesses_time, gp.finished 
FROM game_player gp 
JOIN player p ON gp.player_id = p.id 
WHERE gp.game_id = $1
`

type GamePlayersRow struct {
	ID                 int32
	Username           string
	CorrectGuesses     pgtype.Int4
	CorrectGuessesTime pgtype.Timestamptz
	Finished           pgtype.Timestamptz
}

// returns all the players that played this game
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
			&i.CorrectGuesses,
			&i.CorrectGuessesTime,
			&i.Finished,
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
SELECT g.id, g.creator, g.correct_word, g.created_at, g.started_at, g.ended_at, gp.player_id, gp.played_words, gp.correct_guesses, gp.correct_guesses_time, gp.finished
FROM game g
JOIN game_player gp ON g.id = gp.game_id
WHERE gp.player_id = $1
ORDER BY gp.finished DESC
LIMIT $2 OFFSET $3
`

type PlayerGamesParams struct {
	PlayerID int32
	Limit    int32
	Offset   int32
}

type PlayerGamesRow struct {
	ID                 pgtype.UUID
	Creator            int32
	CorrectWord        string
	CreatedAt          pgtype.Timestamptz
	StartedAt          pgtype.Timestamptz
	EndedAt            pgtype.Timestamptz
	PlayerID           int32
	PlayedWords        []byte
	CorrectGuesses     pgtype.Int4
	CorrectGuessesTime pgtype.Timestamptz
	Finished           pgtype.Timestamptz
}

func (q *Queries) PlayerGames(ctx context.Context, arg PlayerGamesParams) ([]PlayerGamesRow, error) {
	rows, err := q.db.Query(ctx, playerGames, arg.PlayerID, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PlayerGamesRow
	for rows.Next() {
		var i PlayerGamesRow
		if err := rows.Scan(
			&i.ID,
			&i.Creator,
			&i.CorrectWord,
			&i.CreatedAt,
			&i.StartedAt,
			&i.EndedAt,
			&i.PlayerID,
			&i.PlayedWords,
			&i.CorrectGuesses,
			&i.CorrectGuessesTime,
			&i.Finished,
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
