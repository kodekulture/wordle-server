package game

import (
	"iter"
	"maps"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/lordvidex/x/ptr"

	"github.com/kodekulture/wordle-server/game/word"
)

type Response struct {
	CreatedAt       time.Time           `json:"created_at"`
	StartedAt       *time.Time          `json:"started_at"`
	EndedAt         *time.Time          `json:"ended_at"`
	Creator         string              `json:"creator"`
	CorrectWord     *string             `json:"correct_word,omitempty"`     // returned only if game has ended
	Guesses         []GuessResponse     `json:"guesses,omitempty"`          // contains the guesses of the current player
	GamePerformance LeaderboardResponse `json:"game_performance,omitempty"` // contains the best guesses of all players
	ID              uuid.UUID           `json:"id"`
}

type GuessResponse struct {
	// Word can be nil if the word was not played by this user
	Word     *string   `json:"word,omitempty"`
	PlayedAt time.Time `json:"played_at"`
	Status   []int     `json:"status,omitempty"`
}

// PlayerGuessResponse shows the user the effect of the guess he has made on the leaderboard and his rank
type PlayerGuessResponse struct {
	// Result is the result of the guess made by the user
	Result GuessResponse `json:"result"`
	// RankOffset is the amount of players that this user has displaced in the leaderboard
	// this field is set when the game is active, and the user's guess made him move up the leaderboard
	RankOffset *int `json:"rank_offset,omitempty"`
	// Leaderboard contains best guesses for this user.
	Leaderboard LeaderboardResponse `json:"leaderboard"`
}

type LeaderboardResponse []PlayerSummaryResponse

type PlayerSummaryResponse struct {
	// Rank is the position of the user in the leaderboard
	Rank        int           `json:"rank"`
	Best        GuessResponse `json:"best"`
	Username    string        `json:"username"`
	WordsPlayed int           `json:"words_played"`
}

// InitialData is the data sent to the client when a new connection is established
// or when the game is started
type InitialData struct {
	Response
	Active bool `json:"active"`
}

func sorted[T any](x iter.Seq[T], fn func(a, b T) bool) iter.Seq[T] {
	vals := make([]T, 0)
	for v := range x {
		vals = append(vals, v)
	}
	sort.Slice(vals, func(i, j int) bool {
		return fn(vals[i], vals[j])
	})

	return func(yield func(T) bool) {
		for _, v := range vals {
			if !yield(v) {
				return
			}
		}
	}
}

func ToResponse(g Game, username string) Response {
	setWord := func(w string) *string {
		if g.EndedAt == nil {
			return nil
		}
		return ptr.String(w)
	}
	perf := make([]PlayerSummaryResponse, 0, len(g.Sessions))

	var it iter.Seq[*Session]
	if g.IsActive() {
		it = slices.Values(g.Leaderboard.Ranks)
	} else {
		it = sorted(maps.Values(g.Sessions), func(i, j *Session) bool { return i.Player.Username < j.Player.Username })
	}

	for s := range it {
		perf = append(perf, PlayerSummaryResponse{
			Username:    s.Player.Username,
			Best:        ToGuess(s.BestGuess(), false),
			Rank:        g.Leaderboard.Positions[s.Player.Username],
			WordsPlayed: s.WordsCount(),
		})
	}
	var guesses []GuessResponse
	userSession, ok := g.Sessions[username]
	if ok {
		guesses = make([]GuessResponse, len(userSession.Guesses))
		for i, guess := range userSession.Guesses {
			guesses[i] = ToGuess(guess, true)
		}
	}
	return Response{
		CreatedAt:       g.CreatedAt,
		StartedAt:       g.StartedAt,
		EndedAt:         g.EndedAt,
		Creator:         g.Creator,
		CorrectWord:     setWord(g.CorrectWord.Word),
		Guesses:         guesses,
		GamePerformance: perf,
		ID:              g.ID,
	}
}

// ToGuess converts a word.Word to a guessResponse.
// If showWord is true, the word is returned, otherwise it is nil.
func ToGuess(w word.Word, showWord bool) GuessResponse {
	guessed := func() *string {
		if showWord {
			return ptr.String(w.Word)
		}
		return nil
	}
	return GuessResponse{
		Word:     guessed(),
		PlayedAt: w.PlayedAt.Time,
		Status:   w.Stats.Ints(),
	}
}

func ToLeaderboard(l RankBoard) LeaderboardResponse {
	// copy the map, to prevent the original from being modified
	m := make([]PlayerSummaryResponse, len(l.Ranks))
	for i, v := range l.Ranks {
		m[i] = PlayerSummaryResponse{
			Username:    v.Player.Username,
			Best:        ToGuess(v.BestGuess(), false),
			Rank:        l.Positions[v.Player.Username],
			WordsPlayed: v.WordsCount(),
		}
	}
	return LeaderboardResponse(m)
}

// ToInitialData converts a game to initialData for a specific user
// This function is called on game start and on new connection to the game
func ToInitialData(g Game, username string) InitialData {
	return InitialData{
		Response: ToResponse(g, username),
		Active:   g.IsActive(),
	}
}
