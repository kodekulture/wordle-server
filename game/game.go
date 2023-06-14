package game

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/Chat-Map/wordle-server/game/word"
)

const (
	// MaxDuration is the maximum duration a game can last
	MaxDuration = time.Hour

	// MaxGuesses is the maximum number of guesses a player can make
	MaxGuesses = 6

	// WordLength is the length of the word to be guessed
)

type GameStatus int

const (
	Created GameStatus = iota
	Started
	Finished
)

type Game struct {
	CreatedAt   time.Time
	Sessions    map[string]*Session
	StartedAt   *time.Time
	EndedAt     *time.Time
	Creator     string
	CorrectWord word.Word
	finished    int
	ID          uuid.UUID
}

func (g *Game) Start() {
	now := time.Now()
	g.StartedAt = &now
	// Initialize the sessions
}

// Join is used to enter a game before it starts
func (g *Game) Join(username string) {
	g.Sessions[username] = &Session{}
}

// HasEnded returns true if game has ended, otherwise false
func (g *Game) HasEnded() bool {
	return g.EndedAt != nil
}

func New(username string, correctWord word.Word) *Game {
	return &Game{
		ID:          uuid.New(),
		CreatedAt:   time.Now(),
		CorrectWord: correctWord,
		Creator:     username,
		Sessions:    make(map[string]*Session),
	}
}

// Play must be called in a synchronized manner (from a single goroutine) because it modifies the game state
// It returns a boolean indicating whether the guess changed the game state / the session of the player who played the word.
//
// Play also sets the EndTime of the game if the game has ended for every player.
func (g *Game) Play(player string, guess *word.Word) bool {
	session := g.Sessions[player]
	if session == nil {
		return false // TODO: player not found
	}

	if session.Ended() { // game has ended, no need to add more guesses
		return false
	}
	guess.PlayedAt.Scan(time.Now().UTC())
	guess.CompareTo(g.CorrectWord)
	session.Guesses = append(session.Guesses, *guess)
	if guess.Correct() {
		g.finished++
		if g.finished == len(g.Sessions) {
			now := time.Now()
			g.EndedAt = &now // game is over when everyone has finished guessing the word or have failed to guess the word
		}
	}
	return true
}

// Session holds the state of a player's game session.
type Session struct {
	Guesses []word.Word
}

func (s *Session) Latest() word.Word {
	if len(s.Guesses) == 0 {
		return word.Word{}
	}
	return s.Guesses[len(s.Guesses)-1]
}

func (s *Session) JSON() []byte {
	// Error is ignored because we know that the struct is valid
	b, _ := json.Marshal(s.Guesses)
	return b
}

func (s *Session) BestGuess() (w word.Word) {
	var c int
	for _, guess := range s.Guesses {
		v := guess.CorrectCount()
		if v > c {
			c = v
			w = guess
		}
	}
	return w
}

// Won returns true if the last guess is correct
func (s *Session) Won() bool {
	if len(s.Guesses) == 0 {
		return false
	}
	last := s.Guesses[len(s.Guesses)-1]
	return last.Correct()
}

// CanPlay returns true if the user can still play (has not exceeded the maximum number of guesses)
func (s *Session) CanPlay() bool {
	return len(s.Guesses) < MaxGuesses
}

// Ended returns true if the user has finished up all their guesses or they have won the game (guessed the correct word)
func (s *Session) Ended() bool {
	return len(s.Guesses) == MaxGuesses || s.Won()
}
