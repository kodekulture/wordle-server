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
SELECT p.username AS creator_username, g.* from game g
JOIN player p ON game.creator = p.id WHERE g.id = $1;

-- name: FinishGame :exec
UPDATE game SET ended_at = coalesce($2, NOW()) WHERE id = $1;

-- name: CreateGamePlayers :copyfrom
INSERT INTO game_player (game_id, player_id) VALUES ($1, $2);

-- name: UpdateGamePlayer :exec
-- This updates the player stats at the end of the game
UPDATE game_player SET played_words=$3, correct_guesses=$4, correct_guesses_time=$5, finished=$6 
WHERE game_id=$1 AND player_id=$2;

-- name: CreateGame :exec
INSERT INTO game (id, creator, correct_word, created_at, started_at) VALUES ($1, $2, $3, $4, $5);
