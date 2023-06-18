package game

import (
	"reflect"
	"testing"

	"github.com/lordvidex/x/ptr"
	"github.com/stretchr/testify/assert"

	"github.com/Chat-Map/wordle-server/game/word"
)

var words = []word.Word{
	{Word: "HELLO", Stats: word.LetterStatuses{3, 3, 3, 3, 3}},
	{Word: "HALLO", Stats: word.LetterStatuses{3, 1, 3, 3, 3}},
	{
		Word:  "JAMES",
		Stats: word.LetterStatuses{word.Incorrect, 3, word.Incorrect, word.Exists, word.Incorrect},
	},
}

func TestGame_Start(t *testing.T) {
	g := New("test", word.New("SOMEA"))
	g.Start()
	assert.NotNil(t, g.CreatedAt)
}

func TestGame_Play(t *testing.T) {
	testcases := []struct {
		expectErr    error
		g            *Game
		name         string
		player       string
		w            word.Word
		expectStatus word.LetterStatuses
	}{
		{
			name: "player does not exist",
			g: func() *Game {
				g := New("fela", word.New("GAMES"))
				w := word.New("GAMES")
				g.Play("fela", &w)
				return g
			}(),
			w:            word.New("GAMAS"),
			player:       "fela",
			expectErr:    ErrPlayerNotFound,
			expectStatus: word.LetterStatuses{0, 0, 0, 0, 0},
		},
		{
			name: "game has ended",
			g: func() *Game {
				g := New("fela", word.New("GAMES"))
				g.Join(Player{Username: "fela"})
				w := word.New("GAMES")
				g.Play("fela", &w)
				return g
			}(),
			w:            word.New("GAMAS"),
			player:       "fela",
			expectErr:    ErrSessionEnded,
			expectStatus: word.LetterStatuses{0, 0, 0, 0, 0},
		},
		{
			name: "played one word",
			g: func() *Game {
				g := New("fela", word.New("GAMES"))
				g.Join(Player{Username: "fela"})
				return g
			}(),
			w:            word.New("GAMAS"),
			player:       "fela",
			expectErr:    nil,
			expectStatus: word.LetterStatuses{word.Correct, word.Correct, word.Correct, word.Incorrect, word.Correct},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.g
			w := tt.w
			_, result := g.Play(tt.player, &w)
			if tt.expectErr != result {
				t.Errorf("expected %v got %v", tt.expectErr, result)
			}
			if !reflect.DeepEqual(tt.expectStatus, w.Stats) {
				t.Errorf("expected %v got %v", tt.expectStatus, w.Stats)
			}
		})
	}
}

func TestGame_Session(t *testing.T) {
	sessions := []struct {
		s     Session
		name  string
		won   bool
		ended bool
	}{
		{Session{Guesses: nil}, "empty session", false, false},
		{
			s: func() Session {
				wrds := []word.Word{words[1], words[2]}
				s := Session{}
				for _, w := range wrds {
					s.play(w)
				}
				return s
			}(), name: "incomplete session", won: false, ended: false,
		},
		{
			s: func() Session {
				wrds := []word.Word{words[0]}
				s := Session{}
				for _, w := range wrds {
					s.play(w)
				}
				return s
			}(), name: "correct guess", won: true, ended: true,
		},
		{
			Session{Guesses: []word.Word{words[1], words[1], words[2], words[1], words[2], words[2]}},
			"session guesses used up",
			false,
			true,
		},
	}
	for _, tt := range sessions {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.won, tt.s.Won())
			assert.Equal(t, tt.ended, tt.s.Ended())
		})
	}
}

func TestGame_Join(t *testing.T) {
	g := New("test", word.New("SOMEA"))
	g.Join(Player{Username: "fela"})
	assert.Equal(t, 1, len(g.Sessions))
	_, ok := g.Sessions["fela"]
	assert.True(t, ok)
}

func TestRankBoard_FixPosition(t *testing.T) {
	positions := map[string]int{"fela": 0, "james": 1, "jane": 2}
	ranks := []*Session{
		{
			bestGuess: ptr.Obj(words[1]),
			Guesses:   []word.Word{words[1]},
			Player:    Player{Username: "fela"},
		},
		{
			bestGuess: ptr.Obj(words[1]),
			Guesses:   []word.Word{words[1]},
			Player:    Player{Username: "james"},
		},
		{
			bestGuess: ptr.Obj(words[0]),
			Guesses:   []word.Word{words[0]},
			Player:    Player{Username: "jane"},
		},
	}
	type fields struct {
		withPos   func(map[string]int) map[string]int
		withRanks func([]*Session) []*Session
	}
	type args struct {
		username string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "jane should be first",
			fields: fields{
				withPos:   func(p map[string]int) map[string]int { return p },
				withRanks: func(rnks []*Session) []*Session { return rnks },
			},
			args: args{
				username: "jane",
			},
			want: 2,
		},
		{
			name: "james played but didn't move",
			fields: fields{
				withPos: func(p map[string]int) map[string]int {
					p["james"], p["jane"], p["fela"] = 2, 0, 1
					return p
				},
				withRanks: func(rnks []*Session) []*Session {
					rnks[0], rnks[1], rnks[2] = rnks[2], rnks[0], rnks[1]
					return rnks
				},
			},
			args: args{
				username: "james",
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := tt.fields.withPos(copyMap(positions))
			rnks := tt.fields.withRanks(copySlice(copySlice(ranks)))
			r := RankBoard{
				positions: pos,
				ranks:     rnks,
			}
			before := r.positions[tt.args.username]
			got := r.FixPosition(tt.args.username)
			if got != tt.want {
				t.Errorf("RankBoard.FixPosition() = %v, want %v", got, tt.want)
			}
			assert.Equal(t, before-got, r.positions[tt.args.username])
		})
	}
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	c := make(map[K]V)
	for k, v := range m {
		c[k] = v
	}
	return c
}

func copySlice[K any](sl []K) []K {
	c := make([]K, len(sl))
	copy(c, sl)
	return c
}
