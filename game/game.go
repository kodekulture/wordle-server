package game

import (
	"sync"
	"time"

	"github.com/Chat-Map/wordle-server/game/word"

	"github.com/google/uuid"
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

var Hub struct {
	mu    sync.RWMutex // protects the games map
	games map[uuid.UUID]*Game
}

func init() {
	Hub.games = make(map[uuid.UUID]*Game)
}

type Game struct {
	ID          uuid.UUID
	Creator     string              // the username of the player who created the game, only this user can start the game
	CorrectWord word.Word           // the correct word that should be guessed
	Sessions    map[string]*Session // each player's game state, points, position
	finished    int                 // the number of players who have finished the game by guessing the correct word
	CreatedAt   time.Time           // the time the game created
	StartedAt   *time.Time          // the time the game started, when the value is nil, this means the game has not started
	EndedAt     *time.Time          // EndTime is the time the game ended, when the value is nil, this means the game has not ended
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
func (g *Game) Play(player string, guess word.Word) bool {
	session := g.Sessions[player]
	if session == nil {
		return false // TODO: player not found
	}

	if session.Ended() { // game has ended, no need to add more guesses
		return false
	}
	guess.PlayedAt.Scan(time.Now().UTC())
	guess.CompareTo(g.CorrectWord)
	session.Guesses = append(session.Guesses, guess)
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

// Won returns true if the last guess is correct
func (s *Session) Won() bool {
	if len(s.Guesses) == 0 {
		return false
	}
	last := s.Guesses[len(s.Guesses)-1]
	return last.Correct()
}

// Ended returns true if the user has finished up all their guesses or they have won the game (guessed the correct word)
func (s *Session) Ended() bool {
	return len(s.Guesses) == MaxGuesses || s.Won()
}
