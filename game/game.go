package game

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lordvidex/x/ptr"

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

type RankBoard struct {
	ranks     []*Session     // the first element is the highest rank
	positions map[string]int // the current position on the `username` in tha `ranks` array
}

func NewRankBoard(initial map[string]*Session) RankBoard {
	ranks := make([]*Session, 0, len(initial))
	positions := make(map[string]int)
	var index int
	for username, session := range initial {
		ranks = append(ranks, session)
		positions[username] = index
		index++
	}
	return RankBoard{
		ranks:     ranks,
		positions: positions,
	}
}

// FixPosition returns the number of users displaced by the current user.
//
// It should be called after a new guess is made by this user.
func (r RankBoard) FixPosition(username string) int {
	var moves int // the amount of users displaced by the new rank
	index := r.positions[username]
	for i := index; i > 0; i-- {
		curr, prev := r.ranks[i], r.ranks[i-1]
		// TODO: add other comparators in else if
		if curr.GreaterThan(prev) {
			r.ranks[i-1], r.ranks[i] = curr, prev
			r.positions[curr.Player.Username] = i - 1
			r.positions[prev.Player.Username] = i
			moves++
		} else {
			break
		}
	}
	return moves
}

type Game struct {
	CreatedAt time.Time
	Sessions  map[string]*Session
	// There is no leaderboard until the game starts
	Leaderboard RankBoard
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
	g.Leaderboard = NewRankBoard(g.Sessions)
}

// Join is used to enter a game before it starts
func (g *Game) Join(p Player) {
	g.Sessions[p.Username] = &Session{Player: p}
}

// HasEnded returns true if game has ended, otherwise false
func (g *Game) HasEnded() bool {
	return g.EndedAt != nil
}

func New(creator string, correctWord word.Word) *Game {
	return &Game{
		ID:          uuid.New(),
		CreatedAt:   time.Now(),
		CorrectWord: correctWord,
		Creator:     creator,
		Sessions:    make(map[string]*Session),
	}
}

// Play must be called in a synchronized manner (from a single goroutine) because it modifies the game state
// It returns an integer indicating the number of players this user has displaced on the leaderboard.
//
// Play also sets the EndTime of the game if the game has ended for every player.
func (g *Game) Play(player string, guess *word.Word) (int, error) {
	session := g.Sessions[player]
	if session == nil {
		return 0, errors.New("player not found")
	}

	if session.Ended() { // game has ended, no need to add more guesses
		return 0, errors.New("user session has ended")
	}
	// process the guess
	guess.PlayedAt.Scan(time.Now().UTC())
	guess.Check(g.CorrectWord)
	session.play(ptr.ToObj(guess))

	// update the leaderboard
	offset := g.Leaderboard.FixPosition(session.Player.Username)
	if session.Ended() {
		g.finished++
		if g.finished == len(g.Sessions) {
			now := time.Now()
			g.EndedAt = &now // game is over when everyone has finished guessing the word or have failed to guess the word
		}
	}
	return offset, nil
}

func (g *Game) Players() []string {
	usernames := make([]string, len(g.Sessions))
	for username := range g.Sessions {
		usernames = append(usernames, username)
	}
	return usernames
}

// Session holds the state of a player's game session.
type Session struct {
	Guesses   []word.Word
	Player    Player
	bestGuess *word.Word
}

// play updates the current bestGuess made by the user
func (s *Session) play(w word.Word) {
	s.Guesses = append(s.Guesses, w)
	if s.bestGuess == nil {
		s.bestGuess = &w
	} else {
		if w.GreaterThan(s.BestGuess()) {
			s.bestGuess = ptr.Obj(w)
		}
	}
}

func (s *Session) JSON() []byte {
	// Error is ignored because we know that the struct is valid
	b, _ := json.Marshal(s.Guesses)
	return b
}

// BestGuess returns the best guess made by the user
func (s *Session) BestGuess() word.Word {
	return ptr.ToObj(s.bestGuess)
}

func (s *Session) GreaterThan(other *Session) bool {
	return s.BestGuess().GreaterThan(other.BestGuess())
}

// Won returns true if the last guess is correct
func (s *Session) Won() bool {
	if len(s.Guesses) == 0 {
		return false
	}
	return s.BestGuess().Correct()
}

// CanPlay returns true if the user can still play (has not exceeded the maximum number of guesses)
func (s *Session) CanPlay() bool {
	return len(s.Guesses) < MaxGuesses
}

// Ended returns true if the user has finished up all their guesses or they have won the game (guessed the correct word)
func (s *Session) Ended() bool {
	return len(s.Guesses) == MaxGuesses || s.Won()
}

// TODO: it's possible to do later, let's continue; we can just add this to the game maybe when the user is choosing game settings for game mode
// // SessionComparator determines the order of two sessions
// // when the same number of words have been guessed.
// type SessionComparator func(s1, s2 *Session) bool

// var (
// 	ByPlayTime SessionComparator = func(s1, s2 *Session) bool {
// 		return s1.BestGuess().PlayedAt.Time.Before(s2.BestGuess().PlayedAt.Time)
// 	}

// 	ByGuessCount SessionComparator = func(s1, s2 *Session) bool {
// 		return len(s1.Guesses) < len(s2.Guesses)
// 	}
// )
