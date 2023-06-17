package random

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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
			require.Truef(t, ok, "token not found")
			require.Equal(t, tt.username, username, "username not equal")
			require.Equal(t, tt.gameID, gameID, "gameID not equal")
			t.Log("storing twice should not create two token entries")
			token2 := rg.Store(tt.username, tt.gameID)
			require.Equal(t, token, token2, "token should be equal")
			require.Falsef(t, len(rg.s) != 1, "token was stored twice instead of once")
		})
	}
}
