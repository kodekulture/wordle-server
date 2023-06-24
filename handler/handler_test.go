package handler

import (
	"testing"

	"github.com/lordvidex/x/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kodekulture/wordle-server/game"
	"github.com/kodekulture/wordle-server/game/word"
)

func genGame() *game.Game {
	users := []string{"user1", "user2", "user3"}
	g := game.New(users[1], word.New("CORRE"))
	for _, us := range users {
		g.Join(game.Player{Username: us})
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
		g        game.Game
	}
	testcases := []struct {
		name string
		args args
		want game.Response
	}{
		{
			name: "simple session",
			args: args{
				username: "user1",
				g:        *genGame(),
			},
			want: game.Response{
				Guesses: []game.GuessResponse{
					// CORRE
					{Word: ptr.String("NATCO"), Status: []int{1, 1, 1, 2, 2}},
					{Word: ptr.String("NOTCO"), Status: []int{1, 3, 1, 2, 1}},
				},
				GamePerformance: []game.PlayerGuessResponse{
					{
						Username:      "user1",
						GuessResponse: game.GuessResponse{Word: ptr.String("NOTCO"), Status: []int{1, 3, 1, 2, 1}},
					},
					{
						Username:      "user2",
						GuessResponse: game.GuessResponse{Word: ptr.String("NOTCO"), Status: []int{1, 3, 1, 2, 1}},
					},
					{
						Username:      "user3",
						GuessResponse: game.GuessResponse{Word: ptr.String("NOTCO"), Status: []int{1, 3, 1, 2, 1}},
					},
				},
			},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			got := game.ToResponse(tt.args.g, tt.args.username)
			// check that the game.GameResponse sessions are correct
			// compare the game sessions
			require.Equal(t, len(tt.want.Guesses), len(got.Guesses), "sessions length mismatch")
			require.Equal(t, len(tt.want.GamePerformance), len(got.GamePerformance), "game performance length mismatch")

			t.Log("Test Game Guesses for the current user")
			for i, s := range got.Guesses {
				assert.Equal(t, ptr.ToString(tt.want.Guesses[i].Word), ptr.ToString(s.Word), "word mismatch")
				assert.Equal(t, tt.want.Guesses[i].Status, s.Status, "status mismatch")
			}

			t.Log("Test Game ratings for all users")
			m := make(map[string]game.PlayerGuessResponse)
			for _, s := range tt.want.GamePerformance {
				m[s.Username] = s
			}
			for _, s := range got.GamePerformance {
				gott, wantt := s.GuessResponse, m[s.Username].GuessResponse
				assert.Equal(t, wantt.Status, gott.Status, "status mismatch")
				if s.Username != tt.args.username {
					assert.Nil(t, gott.Word, "word should be nil when not the current user")
				}
			}
		})
	}
}
