-- name: AddPlayer :exec
INSERT INTO player (username, password) VALUES ($1, $2);

-- name: GetPlayerByUsername :one
SELECT * FROM player WHERE username = $1;

