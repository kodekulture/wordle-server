package temp

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/Chat-Map/wordle-server/game/word"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/lordvidex/x/ptr"
)

func TestHubRepo(t *testing.T) {
	// create a new HubRepo
	cr := NewHubRepo(testDB)
	type args struct {
		hub map[uuid.UUID]*game.Game
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test1",
			args: args{
				hub: generateRandomHub(),
			},
		},
		{
			name: "test2",
			args: args{
				hub: generateRandomHub(),
			},
		},
		{
			name: "test3",
			args: args{
				hub: generateRandomHub(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cr.Dump(tt.args.hub)
			require.Equal(t, nil, err)
			got, dropFn, err := cr.Load()
			require.Equal(t, nil, err)
			defer func() {
				require.Equal(t, nil, dropFn())
			}()
			assert.Equal(t, nil, err)
			assert.Equal(t, len(tt.args.hub), len(got))
			for k, g2 := range got {
				g1 := tt.args.hub[k]
				require.NotNil(t, g2)
				require.NotNil(t, g1)
				delete(tt.args.hub, k)
				compareGames(t, g1, g2)
			}

		})
	}
}

// compareGames compares two games and their data
func compareGames(t *testing.T, g1 *game.Game, g2 *game.Game) {
	// compare game data
	assert.Equal(t, g1.ID.String(), g2.ID.String())
	assert.Equal(t, g1.Creator, g2.Creator)
	compareTime(t, g1.CreatedAt, g2.CreatedAt)
	compareTime(t, ptr.ToObj(g1.StartedAt), ptr.ToObj(g2.StartedAt))
	compareTime(t, ptr.ToObj(g1.EndedAt), ptr.ToObj(g2.EndedAt))

	// compare correct word
	compareWords(t, &g1.CorrectWord, &g2.CorrectWord)

	// compare players
	assert.Equal(t, len(g1.Players()), len(g2.Players()))
	for i := 0; i < len(g1.Players()); i++ {
		assert.Equal(t, g1.Players()[i], g2.Players()[i])
	}

	// compare sessions
	assert.Equal(t, len(g1.Sessions), len(g2.Sessions))
	for k, s1 := range g1.Sessions {
		// check if session exists in g2
		s2, ok := g2.Sessions[k]
		assert.True(t, ok)
		delete(g2.Sessions, k)
		//compareWords(t, ptr.Obj(s1.BestGuess()), ptr.Obj(s2.BestGuess()))
		assert.Equal(t, len(s1.Guesses), len(s2.Guesses))
		for i := 0; i < len(s1.Guesses); i++ {
			compareWords(t, ptr.Obj(s1.Guesses[i]), ptr.Obj(s2.Guesses[i]))
		}
	}
}

// compareWords compares two words and their playedAt time
func compareWords(t *testing.T, w1 *word.Word, w2 *word.Word) {
	assert.Equal(t, w1.Word, w2.Word)
	assert.InDelta(t, w1.PlayedAt.Time.Second(), w2.PlayedAt.Time.Second(), time.Second.Seconds())
}

func compareTime(t *testing.T, t1 time.Time, t2 time.Time) {
	assert.InDelta(t, t1.Second(), t2.Second(), time.Second.Seconds())
}

func generateRandomHub() map[uuid.UUID]*game.Game {
	hub := make(map[uuid.UUID]*game.Game)
	for i := 0; i < 100; i++ {
		name := gofakeit.Name()
		g := game.New(name, word.New(gofakeit.Country()))
		g.Join(game.Player{Username: name})
		g.Start()
		// play random words
		for i := 0; i < 3; i++ {
			g.Play(name, ptr.Obj(word.New(gofakeit.Country())))
		}
		hub[uuid.New()] = g
	}
	return hub
}
