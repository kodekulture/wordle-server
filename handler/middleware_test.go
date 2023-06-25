package handler

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kodekulture/wordle-server/game"
)

func TestDecodeHeader(t *testing.T) {
	type args struct {
		auth string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "valid token",
			args:    args{fmt.Sprintf("Bearer %s", "valid-token")},
			want:    "valid-token",
			wantErr: false,
		},
		{
			name:    "invalid token",
			args:    args{"Bearer"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid header",
			args:    args{fmt.Sprintf("Bearerinvalid %s", "invalid-token")},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeHeader(tt.args.auth)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("decodeHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlayer(t *testing.T) {
	player := &game.Player{
		Username: "guy",
		Password: "free-guy",
		ID:       123,
	}
	testcases := []struct {
		ctx      context.Context
		expected *game.Player
		name     string
	}{
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: nil,
		},
		{
			name:     "context with no player",
			ctx:      context.WithValue(context.Background(), playerKey, "not a player"),
			expected: nil,
		},
		{
			name:     "context with player",
			ctx:      context.WithValue(context.Background(), playerKey, player),
			expected: player,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			got := Player(tt.ctx)
			assert.Equal(t, tt.expected, got)
		})
	}
}
