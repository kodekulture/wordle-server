-- name: AddPlayer :exec
INSERT INTO player (username, password, session_ts) VALUES ($1, $2, $3);

-- name: FetchPlayerByUsername :one
SELECT * FROM player WHERE username = $1;

-- name: UpdatePlayerSession :exec
UPDATE player SET session_ts = $2 WHERE username = $1;
