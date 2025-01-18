package game

import (
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/lordvidex/x/ptr"

	"github.com/kodekulture/wordle-server/game/word"
)

var (
	ErrPlayerNotFound = errors.New("player not found")
	ErrSessionEnded   = errors.New("user session has ended")
)

const (
	// MaxDuration is the maximum duration a game can last
	MaxDuration = time.Hour

	// MaxGuesses is the maximum number of guesses a player can make
	MaxGuesses = 6
)

type RankBoard struct {
	Positions map[string]int
	Ranks     []*Session
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
		Ranks:     ranks,
		Positions: positions,
	}
}

// Resync ...
func (r RankBoard) Resync() {
	sort.Slice(r.Ranks, func(i, j int) bool {
		return r.Ranks[i].BestGuess().GreaterThan(r.Ranks[j].BestGuess())
	})

	for i, v := range r.Ranks {
		r.Positions[v.Player.Username] = i
	}
}

// FixPosition returns the number of users displaced by the current user.
//
// It should be called after a new guess is made by this user.
func (r RankBoard) FixPosition(username string) int {
	var moves int // the amount of users displaced by the new rank
	index := r.Positions[username]
	for i := index; i > 0; i-- {
		curr, prev := r.Ranks[i], r.Ranks[i-1]
		// TODO: add other comparators in else if
		if curr.GreaterThan(prev) {
			r.Ranks[i-1], r.Ranks[i] = curr, prev
			r.Positions[curr.Player.Username] = i - 1
			r.Positions[prev.Player.Username] = i
			moves++
		} else {
			break
		}
	}
	return moves
}

type Game struct {
	// Sessions and Leaderboard is unserializable to prevent data corruption.
	// They serve as fast access areas for game sessions and can be recomputed from
	// cold/hot storages.
	// TODO: merge Sessions and Leaderboard
	Sessions    map[string]*Session
	Leaderboard RankBoard `json:"-"`

	CreatedAt   time.Time
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

// IsActive returns true if game has started, otherwise false
func (g Game) IsActive() bool {
	return g.StartedAt != nil
}

// HasEnded returns true if game has ended, otherwise false
func (g *Game) HasEnded() bool {
	return g.EndedAt != nil
}

// New should only be called for new games
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
func (g *Game) Play(player string, guess *word.Word) (int, bool, error) {
	session := g.Sessions[player]
	if session == nil {
		return 0, false, ErrPlayerNotFound
	}

	if session.Ended() { // game has ended, no need to add more guesses
		return 0, false, ErrSessionEnded
	}
	// process the guess
	guess.PlayedAt.Scan(time.Now().UTC())
	guess.Check(g.CorrectWord)
	usersBest := session.play(ptr.ToObj(guess))

	// update the leaderboard
	offset := g.Leaderboard.FixPosition(session.Player.Username)
	if session.Ended() {
		g.finished++
		if g.finished == len(g.Sessions) {
			now := time.Now()
			g.EndedAt = &now // game is over when everyone has finished guessing the word or have failed to guess the word
		}
	}
	return offset, usersBest, nil
}

// Resync ...
func (g *Game) Resync() {
	for _, session := range g.Sessions {
		session.Resync()
		if session.Won() {
			g.finished++
		}
	}
	g.Leaderboard.Resync()
}

func (g *Game) Players() []string {
	usernames := make([]string, 0, len(g.Sessions))
	for username := range g.Sessions {
		usernames = append(usernames, username)
	}
	return usernames
}

// Session holds the state of a player's game session.
type Session struct {
	bestGuess *word.Word
	// the number of words this player has guessed for finished games. It is zero when guesses is empty
	wordsCount int
	Player     Player
	Guesses    []word.Word
}

// SetWordsCount ...
func (s *Session) SetWordsCount(x int) {
	s.wordsCount = x
}

// Resync loops over the player's guesses and updates the best guess
func (s *Session) Resync() {
	wrds := s.Guesses
	s.Guesses = nil
	s.bestGuess = nil
	for _, w := range wrds {
		s.play(w)
	}
}

// play updates the current bestGuess made by the user
func (s *Session) play(w word.Word) bool {
	s.Guesses = append(s.Guesses, w)
	if s.bestGuess == nil {
		s.bestGuess = &w
		return true
	}
	if w.GreaterThan(s.BestGuess()) {
		s.bestGuess = ptr.Obj(w)
		return true
	}

	return false
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

func (s *Session) WordsCount() int {
	if s.wordsCount != 0 {
		return s.wordsCount
	}
	return len(s.Guesses)
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
