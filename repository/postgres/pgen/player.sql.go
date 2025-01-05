// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: player.sql

package pgen

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const addPlayer = `-- name: AddPlayer :exec
INSERT INTO player (username, password, session_ts) VALUES ($1, $2, $3)
`

type AddPlayerParams struct {
	Username  string
	Password  string
	SessionTs pgtype.Int8
}

func (q *Queries) AddPlayer(ctx context.Context, arg AddPlayerParams) error {
	_, err := q.db.Exec(ctx, addPlayer, arg.Username, arg.Password, arg.SessionTs)
	return err
}

const fetchPlayerByUsername = `-- name: FetchPlayerByUsername :one
SELECT id, username, password, session_ts FROM player WHERE username = $1
`

func (q *Queries) FetchPlayerByUsername(ctx context.Context, username string) (Player, error) {
	row := q.db.QueryRow(ctx, fetchPlayerByUsername, username)
	var i Player
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Password,
		&i.SessionTs,
	)
	return i, err
}

const updatePlayerSession = `-- name: UpdatePlayerSession :exec
UPDATE player SET session_ts = $2 WHERE username = $1
`

type UpdatePlayerSessionParams struct {
	Username  string
	SessionTs pgtype.Int8
}

func (q *Queries) UpdatePlayerSession(ctx context.Context, arg UpdatePlayerSessionParams) error {
	_, err := q.db.Exec(ctx, updatePlayerSession, arg.Username, arg.SessionTs)
	return err
}
