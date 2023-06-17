package random

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestRandomGenStore(t *testing.T) {
	tests := []struct {
		name     string
		username string
		gameID   uuid.UUID
	}{
		{
			name:     "test1",
			username: "test1",
			gameID:   uuid.New(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := New(context.TODO())
			token := rg.Store(tt.username, tt.gameID)
			username, gameID, ok := rg.Get(token)
			if !ok {
				t.Errorf("token not found")
			}
			if username != tt.username {
				t.Errorf("username not equal")
			}
			if gameID != tt.gameID {
				t.Errorf("gameID not equal")
			}
			t.Log("storing twice should not create two token entries")
			token2 := rg.Store(tt.username, tt.gameID)
			if token != token2 {
				t.Errorf("token should be equal")
			}
			if len(rg.s) != 1 {
				t.Errorf("token was stored twice instead of once")
			}
		})
	}
}
