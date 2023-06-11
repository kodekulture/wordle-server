-- name: PlayerGames :many
SELECT g.*, gp.player_id, gp.played_words, gp.correct_guesses, gp.correct_guesses_time, gp.finished
FROM game g
JOIN game_player gp ON g.id = gp.game_id
WHERE gp.player_id = $1
ORDER BY gp.finished DESC
LIMIT $2 OFFSET $3;

-- name: GamePlayers :many
-- returns all the players that played this game
SELECT p.id, p.username, gp.correct_guesses, gp.correct_guesses_time, gp.finished 
FROM game_player gp 
JOIN player p ON gp.player_id = p.id 
WHERE gp.game_id = $1;
