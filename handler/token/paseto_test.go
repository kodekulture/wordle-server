package token

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Chat-Map/wordle-server/game"
)

func TestNew(t *testing.T) {
	type args struct {
	footer   string
	key      []byte
	validity time.Duration
}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid key len",
			args: args{
				key:      []byte("12345678901234567890123456789012"),
				footer:   "footer",
				validity: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "invalid key len",
			args: args{
				key:      []byte("key"),
				footer:   "footer",
				validity: 24 * time.Hour,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.args.key, tt.args.footer, tt.args.validity)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestPasetoGenerate(t *testing.T) {
	p := newPasetoTest(t)
	type args struct {
		player game.Player
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid generate",
			args: args{player: game.Player{
				Password: "password",
				Username: "username",
				ID:       1,
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Generate(context.Background(), tt.args.player)
			if (err != nil) != tt.wantErr {
				t.Errorf("Paseto.Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestPasetoValidate(t *testing.T) {
	p := newPasetoTest(t)
	type args struct {
		player game.Player
	}
	tests := []struct {
		name    string
		args    args
		want    game.Player
		wantErr bool
	}{
		{
			name: "valid generate",
			args: args{player: game.Player{
				Password: "password",
				Username: "username",
				ID:       1,
			}},
			want: game.Player{
				Password: "",
				Username: "username",
				ID:       1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := p.Generate(context.Background(), tt.args.player)
			if (err != nil) != tt.wantErr {
				t.Errorf("Paseto.Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, err := p.Validate(context.Background(), token)
			if (err != nil) != tt.wantErr {
				t.Errorf("Paseto.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Paseto.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// newPasetoTest creates a new paseto instance for testing purposes
func newPasetoTest(t *testing.T) *Paseto {
	p, err := New([]byte("12345678901234567890123456789012"), "footer", 24*time.Hour)
	if err != nil {
		t.Errorf("Failed to create paseto: %v", err)
	}
	return p
}
