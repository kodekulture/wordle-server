-- name: PlayerGames :many
SELECT g.id, g.correct_word, g.created_at, g.started_at, g.ended_at, 
  p.id AS creator_id, p.username AS creator_username,
  gp.player_id, gp.played_words, gp.correct_guesses, gp.correct_guesses_time, gp.finished
FROM game g
JOIN game_player gp ON g.id = gp.game_id
JOIN player p ON g.creator = p.id
WHERE gp.player_id = $1
ORDER BY gp.finished DESC
LIMIT sqlc.narg('limit') OFFSET $2;

-- name: GamePlayers :many
-- returns all the players that played this game
SELECT p.id, p.username, gp.correct_guesses, gp.correct_guesses_time, gp.finished 
FROM game_player gp 
JOIN player p ON gp.player_id = p.id 
WHERE gp.game_id = $1;

-- name: FetchGame :one
SELECT * from game WHERE id = $1;

-- name: StartGame :exec
UPDATE game SET started_at = coalesce($2, NOW()) WHERE id = $1;

-- name: FinishGame :exec
UPDATE game SET ended_at = coalesce($2, NOW()) WHERE id = $1;

-- name: UpdatePlayerStats :exec
-- This upserts the player stats if they were not already present
INSERT INTO game_player (game_id, player_id) VALUES ($1, $2)
ON CONFLICT (game_id, player_id) 
DO UPDATE SET played_words=$3, correct_guesses=$4, correct_guesses_time=$5, finished=$6;

-- name: CreateGame :exec
INSERT INTO game (id, creator, correct_word) VALUES ($1, $2, $3);