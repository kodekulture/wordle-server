package random

import (
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
			rg := New()
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
		})
	}
}
