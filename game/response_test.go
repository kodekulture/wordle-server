package game

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lordvidex/x/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kodekulture/wordle-server/game/word"
)

func sampleGame() Game {
	sampleTime := time.Time{}
	sessions := []*Session{
		{
			Player:    Player{Username: "test"},
			Guesses:   []word.Word{words[2], words[1]},
			bestGuess: ptr.Obj(words[1]),
		},
		{
			Player:    Player{Username: "second_test"},
			Guesses:   []word.Word{},
			bestGuess: nil,
		},
	}
	m := make(map[string]*Session)
	for _, s := range sessions {
		m[s.Player.Username] = s
	}
	g := Game{
		Sessions: m,
		Leaderboard: RankBoard{
			Positions: map[string]int{"test": 0, "second_test": 1},
			Ranks:     sessions,
		},
		StartedAt:   ptr.Obj(sampleTime),
		CorrectWord: words[0],
	}
	return g
}

func ExampleToResponse() {
	response := ToResponse(sampleGame(), "test")
	var b strings.Builder
	bytes, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	b.Write(bytes)
	fmt.Println(b.String())
	// Output:{"created_at":"0001-01-01T00:00:00Z","started_at":"0001-01-01T00:00:00Z","ended_at":null,"creator":"","guesses":[{"word":"JAMES","played_at":"0001-01-01T00:00:00Z","status":[1,3,1,2,1]},{"word":"HALLO","played_at":"0001-01-01T00:00:00Z","status":[3,1,3,3,3]}],"game_performance":[{"rank":0,"best":{"played_at":"0001-01-01T00:00:00Z","status":[3,1,3,3,3]},"username":"test","words_played":2},{"rank":1,"best":{"played_at":"0001-01-01T00:00:00Z"},"username":"second_test","words_played":0}],"id":"00000000-0000-0000-0000-000000000000"}
}

func TestToGuess(t *testing.T) {
	type args struct {
		w        word.Word
		showWord bool
	}
	tests := []struct {
		name string
		args args
		want GuessResponse
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToGuess(tt.args.w, tt.args.showWord); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToGuess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func genGame() *Game {
	users := []string{"user1", "user2", "user3"}
	g := New(users[1], word.New("CORRE"))
	for _, us := range users {
		g.Join(Player{Username: us})
	}
	for _, us := range users {
		w := word.New("NATCO")
		w2 := word.New("NOTCO")
		g.Play(us, &w)
		g.Play(us, &w2)
	}
	return g
}

func TestToGame(t *testing.T) {
	type args struct {
		username string
		g        Game
	}
	testcases := []struct {
		name string
		args args
		want Response
	}{
		{
			name: "simple session",
			args: args{
				username: "user1",
				g:        *genGame(),
			},
			want: Response{
				Guesses: []GuessResponse{
					// CORRE
					{Word: ptr.String("NATCO"), Status: []int{1, 1, 1, 2, 2}},
					{Word: ptr.String("NOTCO"), Status: []int{1, 3, 1, 2, 1}},
				},
				GamePerformance: []PlayerSummaryResponse{
					{
						Username:    "user1",
						Rank:        1,
						WordsPlayed: 2,
						Best:        GuessResponse{Word: ptr.String("NOTCO"), Status: []int{1, 3, 1, 2, 1}},
					},
					{
						Username:    "user2",
						Rank:        2,
						WordsPlayed: 2,
						Best:        GuessResponse{Word: ptr.String("NOTCO"), Status: []int{1, 3, 1, 2, 1}},
					},
					{
						Username:    "user3",
						Rank:        3,
						WordsPlayed: 2,
						Best:        GuessResponse{Word: ptr.String("NOTCO"), Status: []int{1, 3, 1, 2, 1}},
					},
				},
			},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			got := ToResponse(tt.args.g, tt.args.username)
			// check that the GameResponse sessions are correct
			// compare the sessions
			require.Equal(t, len(tt.want.Guesses), len(got.Guesses), "sessions length mismatch")
			require.Equal(t, len(tt.want.GamePerformance), len(got.GamePerformance), "game performance length mismatch")

			t.Log("Test Game Guesses for the current user")
			for i, s := range got.Guesses {
				assert.Equal(t, ptr.ToString(tt.want.Guesses[i].Word), ptr.ToString(s.Word), "word mismatch")
				assert.Equal(t, tt.want.Guesses[i].Status, s.Status, "status mismatch")
			}

			t.Log("Test Game ratings for all users")
			m := make(map[string]PlayerSummaryResponse)
			for _, s := range tt.want.GamePerformance {
				m[s.Username] = s
			}
			for _, s := range got.GamePerformance {
				gott, wantt := s.Best, m[s.Username].Best
				assert.Equal(t, wantt.Status, gott.Status, "status mismatch")
				if s.Username != tt.args.username {
					assert.Nil(t, gott.Word, "word should be nil when not the current user")
				}
			}
		})
	}
}

func TestToInitialData(t *testing.T) {
	type args struct {
		username string
		g        Game
	}
	tests := []struct {
		name string
		want InitialData
		args args
	}{
		{
			name: "inactive game",
			args: args{
				g: Game{
					Sessions: map[string]*Session{
						"test": {Player: Player{Username: "test"}},
					},
				},
				username: "test",
			},
			want: InitialData{
				Response: Response{
					Guesses: []GuessResponse{},
					GamePerformance: LeaderboardResponse{
						{
							Best:     GuessResponse{Status: []int{}},
							Username: "test",
						},
					},
				},
				Active: false,
			},
		},
		{
			name: "active game",
			args: args{
				g:        sampleGame(),
				username: "test",
			},
			want: InitialData{
				Active: true,
				Response: Response{
					Guesses: []GuessResponse{
						{Word: ptr.String(words[2].String()), Status: words[2].Stats.Ints()},
						{Word: ptr.String(words[1].String()), Status: words[1].Stats.Ints()},
					},
					GamePerformance: LeaderboardResponse{
						{Rank: 0, Best: GuessResponse{Status: []int{3, 1, 3, 3, 3}}, Username: "test", WordsPlayed: 2},
						{Rank: 1, Best: GuessResponse{Status: []int{}}, Username: "second_test"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToInitialData(tt.args.g, tt.args.username)
			assert.Equal(t, tt.want.GamePerformance, got.GamePerformance)
			assert.Equal(t, tt.want.Active, got.Active)
			assert.Equal(t, tt.want.Guesses, got.Guesses)

		})
	}
}
