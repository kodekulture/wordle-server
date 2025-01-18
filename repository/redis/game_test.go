package redis

import (
	"encoding/json"
	"testing"

	"github.com/kodekulture/wordle-server/game"
	"github.com/kodekulture/wordle-server/game/word"
)

func TestGame_Marshal(t *testing.T) {
	g := game.New("test", word.New("EVADE"))
	players := []string{"user1", "user2", "user3", "user4", "user5"}
	for i, player := range players {
		g.Join(game.Player{Username: player, ID: i})
	}

	rg := Game{Game: g}
	b, _ := json.Marshal(rg)
	t.Log(string(b))
}
