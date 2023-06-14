package game

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Chat-Map/wordle-server/game/word"
)

func TestGame_Start(t *testing.T) {
	g := New("test", word.New("SOMEA"))
	g.Start()
	assert.NotNil(t, g.CreatedAt)
}

func TestGame_Play(t *testing.T) {
	testcases := []struct {
		g            *Game
		name         string
		player       string
		w            word.Word
		expectStatus []word.LetterStatus
		expect       bool
	}{
		{
			name: "game has ended",
			g: func() *Game {
				g := New("fela", word.New("GAMES"))
				w := word.New("GAMES")
				w.Stats = w.CompareTo(w)
				g.Sessions["fela"] = &Session{Guesses: []word.Word{w}}
				return g
			}(),
			w:            word.New("GAMAS"),
			player:       "fela",
			expect:       false,
			expectStatus: []word.LetterStatus{0, 0, 0, 0, 0},
		},
		{
			name: "played one word",
			g: func() *Game {
				g := New("fela", word.New("GAMES"))
				g.Sessions["fela"] = &Session{Guesses: []word.Word{}}
				return g
			}(),
			w:            word.New("GAMAS"),
			player:       "fela",
			expect:       true,
			expectStatus: []word.LetterStatus{word.Correct, word.Correct, word.Correct, word.Incorrect, word.Correct},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.g
			w := tt.w
			result := g.Play(tt.player, &w)
			if tt.expect != result {
				t.Errorf("expected %v got %v", tt.expect, result)
			}
			if !reflect.DeepEqual(tt.expectStatus, w.Stats) {
				t.Errorf("expected %v got %v", tt.expectStatus, w.Stats)
			}
		})
	}
}

func TestGame_Session(t *testing.T) {
	words := []word.Word{
		{Word: "HELLO", Stats: []word.LetterStatus{3, 3, 3, 3, 3}},
		{Word: "HALLO", Stats: []word.LetterStatus{3, 1, 3, 3, 3}},
		{
			Word:  "JAMES",
			Stats: []word.LetterStatus{word.Incorrect, 3, word.Incorrect, word.Exists, word.Incorrect},
		},
	}
	sessions := []struct {
		s     Session
		name  string
		won   bool
		ended bool
	}{
		{Session{Guesses: nil}, "empty session", false, false},
		{Session{Guesses: []word.Word{words[1], words[2]}}, "incomplete session", false, false},
		{Session{Guesses: []word.Word{words[0]}}, "won", true, true},
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
	g.Join("fela")
	assert.Equal(t, 1, len(g.Sessions))
	_, ok := g.Sessions["fela"]
	assert.True(t, ok)
}
