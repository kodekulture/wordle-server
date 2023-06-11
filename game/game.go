package game

import (
	"sort"
	"time"
	"wordle/cmd/game/word"

	"github.com/google/uuid"
)

const (
	// MaxDuration is the maximum duration a game can last
	MaxDuration = time.Hour

	// MaxGuesses is the maximum number of guesses a player can make
	MaxGuesses = 6

	// WordLength is the length of the word to be guessed
)

// Player is the interface that represents a player in the game
type Player interface {
	PID() string
	PName() string
}

type Game struct {
	ID          uuid.UUID
	CreatorID   string     // the id of the player who created the game, this is low priority for now
	InviteID    string     // the id of the lobby used to play this game
	CorrectWord word.Word  // the correct word that should be guessed
	Sessions    []*Session // each player's game state, points, position
	PlayerCount int        // the number of players in the game
	finished    int        // the number of players who have finished the game by guessing the correct word
	StartTime   time.Time  // the time the game started

	// EndTime is the time the game ended,
	// when the value is nil, this means the game has not ended
	EndTime *time.Time
}

func New(roomID string, players []Player, correctWord word.Word) *Game {
	sessions := make([]*Session, len(players))
	for i, p := range players {
		sessions[i] = NewSession(p)
	}
	return &Game{
		ID:          uuid.New(),
		StartTime:   time.Now(),
		CorrectWord: correctWord,
		InviteID:    roomID,
		PlayerCount: len(players),
		Sessions:    sessions,
	}
}

// Play must be called in a synchronized manner (from a single goroutine) because it modifies the game state
// It returns a boolean indicating whether the guess changed the game state / the session of the player who played the word.
//
// Play also sets the EndTime of the game if the game has ended for every player.
func (g *Game) Play(player Player, guess word.Word) bool {
	for _, s := range g.Sessions {
		if s.Player.PID() == player.PID() { // found the player
			if s.Ended() { // game has ended, no need to add more guesses
				return false
			}
			guess.PlayedAt.Scan(time.Now().UTC())
			guess.CompareTo(g.CorrectWord)
			s.Guesses = append(s.Guesses, guess)
			if guess.Correct() {
				g.finished++
				if g.finished == g.PlayerCount {
					now := time.Now()
					g.EndTime = &now // game is over when everyone has finished guessing the word or have failed to guess the word
				}
			}
			return true
		}
	}
	return false // player not found
}

// TODO: implement this
func (g *Game) Result() []Player {
	sort.Slice(g.Sessions, func(i, j int) bool {
		a, b := g.Sessions[i], g.Sessions[j]
		if a.Won() && !b.Won() {
			return true
		} else if !a.Won() && b.Won() {
			return false
		} else {
			return len(a.Guesses) < len(b.Guesses)
		}
	})
	players := make([]Player, len(g.Sessions))
	for i, s := range g.Sessions {
		players[i] = s.Player
	}
	return players
}

// HasEnded Game if the Game.EndTime is set OR if the game has been active for an hour
// Ended games do not receive rewards after completed Sessions and penalties are applied
// to all sessions immediately after Game has ended.
// or if they have guessed the word correctly
func (g *Game) HasEnded() bool {
	return g.EndTime != nil && g.EndTime.After(g.StartTime.Add(MaxDuration))
}

// Session holds the state of a player's game session.
type Session struct {
	Player  Player
	Guesses []word.Word
}

func NewSession(player Player) *Session {
	return &Session{
		Player: player,
	}
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

// TODO: replace []*Session with PlayerSessions that uses a heap to keep track of the player with the most points
