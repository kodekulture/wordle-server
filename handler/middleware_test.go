package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lordvidex/x/auth"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/kodekulture/wordle-server/game"
	"github.com/kodekulture/wordle-server/internal/mocks"
)

func TestSessionMiddleware(t *testing.T) {
	tests := []struct {
		name                 string
		reqCookies           []http.Cookie
		expectCode           int
		expectExpiredCookies bool
		mockFn               func(*mocks.MockService, *mocks.MockTokenHandler)
	}{
		{
			name: "valid access and refresh tokens",
			reqCookies: []http.Cookie{
				newAccessCookie("valid_access"),
				newRefreshCookie("valid_refresh"),
			},
			mockFn: func(srv *mocks.MockService, th *mocks.MockTokenHandler) {
				th.EXPECT().
					Validate(gomock.Any(), auth.Token("valid_access")).
					Return(game.Player{Username: "test"}, nil)
				srv.EXPECT().GetPlayer(gomock.Any(), "test").
					Return(&game.Player{Username: "test"}, nil)
			},
			expectCode: http.StatusOK,
		},
		{
			name: "valid refresh tokens resets access token if empty",
			reqCookies: []http.Cookie{
				newRefreshCookie("valid_refresh"),
			},
			mockFn: func(srv *mocks.MockService, th *mocks.MockTokenHandler) {
				gomock.InOrder(
					th.EXPECT().
						Validate(gomock.Any(), auth.Token("valid_refresh")).
						Return(game.Player{Username: "test"}, nil),
					srv.EXPECT().GetPlayer(gomock.Any(), "test").
						Return(&game.Player{Username: "test"}, nil),
					th.EXPECT().
						Generate(gomock.Any(), game.Player{Username: "test"}, accessTokenTTL).
						Return("valid_access", nil),
				)

			},
			expectCode: http.StatusOK,
		},
		{
			name: "valid tokens in cookies return unauthenticated response because session has been reset",
			reqCookies: []http.Cookie{
				newAccessCookie("valid_access"),
			},
			mockFn: func(srv *mocks.MockService, th *mocks.MockTokenHandler) {
				th.EXPECT().Validate(gomock.Any(), auth.Token("valid_access")).
					Return(game.Player{
						Username:  "test",
						SessionTs: time.Now().Add(-time.Hour * 24).Unix(),
					}, nil)
				srv.EXPECT().GetPlayer(gomock.Any(), "test").
					Return(&game.Player{
						Username:  "test",
						SessionTs: time.Now().Unix(),
					}, nil)
			},
			expectCode: http.StatusUnauthorized,
		},
		{
			name:       "empty cookies return unauthenticated response",
			expectCode: http.StatusUnauthorized,
		},
		{
			name: "valid refresh tokens resets access token if expired",
			reqCookies: []http.Cookie{
				newAccessCookie("expired_access"),
				newRefreshCookie("valid_refresh"),
			},
			mockFn: func(srv *mocks.MockService, th *mocks.MockTokenHandler) {
				th.EXPECT().
					Validate(gomock.Any(), auth.Token("expired_access")).
					Return(game.Player{}, errors.New("token invalid"))
				th.EXPECT().
					Validate(gomock.Any(), auth.Token("valid_refresh")).
					Return(game.Player{Username: "test"}, nil)
				srv.EXPECT().GetPlayer(gomock.Any(), "test").
					Return(&game.Player{Username: "test"}, nil)
				th.EXPECT().Generate(gomock.Any(), game.Player{Username: "test"}, accessTokenTTL).
					Return("valid_access", nil)
			},
			expectCode: http.StatusOK,
		},
		{
			name: "when valid cookie with invalid refresh token is passed, unauthenticated error is returned and cookie gets invalidated",
			reqCookies: []http.Cookie{
				newRefreshCookie("expired_refresh"),
			},
			mockFn: func(_ *mocks.MockService, th *mocks.MockTokenHandler) {
				th.EXPECT().
					Validate(gomock.Any(), auth.Token("expired_refresh")).
					Return(game.Player{}, errors.New("expired"))
			},
			expectCode:           http.StatusUnauthorized,
			expectExpiredCookies: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := mocks.NewMockService(ctrl)
			th := mocks.NewMockTokenHandler(ctrl)

			if tt.mockFn != nil {
				tt.mockFn(srv, th)
			}

			h := New(srv, th)
			req := httptest.NewRequest("GET", "/me", nil)
			for _, ck := range tt.reqCookies {
				req.AddCookie(&ck)
			}
			recorder := httptest.NewRecorder()

			protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.NotNil(t, Player(r.Context()))
				w.WriteHeader(http.StatusOK)
			})

			handler := h.sessionMiddleware(protected)
			handler.ServeHTTP(recorder, req)

			// assert
			assert.Equal(t, tt.expectCode, recorder.Code)
			if tt.expectExpiredCookies {
				cookies := recorder.Result().Cookies()
				for _, ck := range cookies {
					assert.True(t, time.Now().After(ck.Expires) || ck.MaxAge <= 0)
				}
			}
		})
	}
}

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
