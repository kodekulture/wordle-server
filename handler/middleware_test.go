package handler

import (
	"fmt"
	"testing"
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
