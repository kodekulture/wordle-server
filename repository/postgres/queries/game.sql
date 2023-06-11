-- name: PlayerGames :many
SELECT g.id, g.correct_word, g.created_at, g.started_at, g.ended_at, 
  p.id AS creator_id, p.username AS creator_username,
  gp.player_id, gp.played_words, gp.correct_guesses, gp.correct_guesses_time, gp.finished
FROM game g
JOIN game_player gp ON g.id = gp.game_id
WHERE gp.player_id = $1
JOIN player p ON g.creator = p.id
ORDER BY gp.finished DESC
LIMIT sqlc.narg('limit') OFFSET $2;

-- name: GamePlayers :many
-- returns all the players that played this game
SELECT p.id, p.username, gp.correct_guesses, gp.correct_guesses_time, gp.finished 
FROM game_player gp 
JOIN player p ON gp.player_id = p.id 
WHERE gp.game_id = $1;
