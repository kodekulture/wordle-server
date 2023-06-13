package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/lordvidex/errs"
	"github.com/lordvidex/x/auth"
	"github.com/lordvidex/x/resp"

	"github.com/Chat-Map/wordle-server/game"
)

const (
	authHeaderKey = "Authorization"
)

type (
	contextKey struct {
		name string
	}
	AuthDecodeType int
)

const (
	// AuthDecodeTypeNone indicates that the auth middleware returns the decoded player untouched
	AuthDecodeTypeNone AuthDecodeType = iota
	// AuthDecodeTypeFetch indicates that the auth middleware fetches the player from the database
	AuthDecodeTypeFetch
)

// private vars
var (
	playerKey = &contextKey{"player"}
)

// Errors
var (
	ErrUnauthenticated = errs.B().Code(errs.Unauthenticated).Msg("user is unauthenticated").Err()
)

func PlayerFromCtx(ctx context.Context) *game.Player {
	v, _ := ctx.Value(playerKey).(*game.Player)
	return v
}

// authMiddleware extracts the token from the authorization header of the request
// validates it, and returns a new context that contains the player object.
//
// The injected player can be gotten with the function Player.
func (h *Handler) authMiddleware(fetchType AuthDecodeType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			authHeader := r.Header.Get(authHeaderKey)
			token, err := decodeHeader(authHeader)
			if err != nil {
				resp.Error(w, err)
				return
			}
			player, err := h.token.Validate(ctx, auth.Token(token))
			if err != nil {
				resp.Error(w, ErrUnauthenticated)
				return
			}
			if fetchType == AuthDecodeTypeFetch {
				var temp *game.Player
				if temp, err = h.srv.GetPlayer(ctx, player.Username); err != nil {
					resp.Error(w, err)
					return
				}
				player = *temp
			}
			// replace the request context
			ctx = context.WithValue(ctx, playerKey, &player)
			r = r.WithContext(ctx)

			// pass to the next handler
			next.ServeHTTP(w, r)
		})
	}
}

func decodeHeader(auth string) (string, error) {
	spl := strings.Split(auth, " ")
	switch len(spl) {
	case 1:
		return spl[0], nil
	case 2:
		if strings.ToLower(spl[0]) != "bearer" {
			return "", ErrUnauthenticated
		}
		return spl[1], nil
	default:
		return "", ErrUnauthenticated
	}
}
