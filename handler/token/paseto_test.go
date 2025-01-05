package token

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/kodekulture/wordle-server/game"
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
				key:    []byte("12345678901234567890123456789012"),
				footer: "footer",
			},
			wantErr: false,
		},
		{
			name: "invalid key len",
			args: args{
				key:    []byte("key"),
				footer: "footer",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.args.key, tt.args.footer)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestPasetoGenerate(t *testing.T) {
	type fields struct {
		key      []byte
		footer   string
		validity time.Duration
	}
	type args struct {
		player game.Player
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "valid generate",
			args: args{player: game.Player{
				Password: "password",
				Username: "username",
				ID:       1,
			}},
			fields: fields{
				key:      []byte("12345678901234567890123456789012"),
				footer:   "footer",
				validity: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "invalid generate key",
			args: args{player: game.Player{
				Password: "password",
				Username: "username",
				ID:       1,
			}},
			fields: fields{
				key:      []byte("1234567890123456789012345678901"),
				footer:   "footer",
				validity: 24 * time.Hour,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Paseto{symmetricKey: tt.fields.key, footer: tt.fields.footer}
			_, err := p.Generate(context.Background(), tt.args.player, tt.fields.validity)
			if (err != nil) != tt.wantErr {
				t.Errorf("Paseto.Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestPasetoValidate(t *testing.T) {
	type fields struct {
		key    []byte
		footer string
	}
	type args struct {
		player   game.Player
		validity time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    game.Player
		wantErr bool
	}{
		{
			name: "valid",
			args: args{player: game.Player{
				Password: "password",
				Username: "username",
				ID:       1,
			},
				validity: 24 * time.Hour,
			},
			fields: fields{
				key:    []byte("12345678901234567890123456789012"),
				footer: "",
			},
			want: game.Player{
				Password: "",
				Username: "username",
				ID:       1,
			},
			wantErr: false,
		},
		{
			name: "expired",
			args: args{player: game.Player{
				Password: "password",
				Username: "username",
				ID:       1,
			},
				validity: -24 * time.Hour,
			},
			fields: fields{
				key:    []byte("12345678901234567890123456789012"),
				footer: "footer",
			},
			want: game.Player{
				Password: "",
				Username: "username",
				ID:       1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Paseto{symmetricKey: tt.fields.key, footer: tt.fields.footer}
			token, _ := p.Generate(context.Background(), tt.args.player, tt.args.validity)
			got, err := p.Validate(context.Background(), token)
			if (err != nil) != tt.wantErr {
				t.Errorf("Paseto.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Paseto.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
