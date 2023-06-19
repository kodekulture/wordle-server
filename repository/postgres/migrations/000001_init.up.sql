CREATE TABLE IF NOT EXISTS player (
  id SERIAL PRIMARY KEY,
  username VARCHAR(255) NOT NULL UNIQUE,
  password VARCHAR(254) NOT NULL
);

CREATE TABLE IF NOT EXISTS game (
  id UUID PRIMARY KEY,
  creator INTEGER NOT NULL REFERENCES player(id),
  correct_word VARCHAR(10) NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  started_at TIMESTAMPTZ,
  ended_at TIMESTAMPTZ
);

-- game<->player
CREATE TABLE IF NOT EXISTS game_player (
  game_id UUID NOT NULL REFERENCES game(id),
  player_id INTEGER NOT NULL REFERENCES player(id),
  --json data containing list of words played by this user (should only be shown to the user who owns this data)
  played_words JSONB,
  -- how many letters this user was able to guess correctly
  best_guess varchar(10),
  -- time taken to get his correct_guesses
  best_guess_time TIMESTAMPTZ,
  -- 
  -- time he finished the game -- when null, this user is still playing
  finished TIMESTAMPTZ,

  rank INTEGER, -- the position of this player in the game
  PRIMARY KEY (game_id, player_id)
);

