package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/lordvidex/errs/v2"
	"github.com/lordvidex/x/auth"
	"github.com/lordvidex/x/resp"

	"github.com/kodekulture/wordle-server/game"
)

const (
	accessTokenKey  = "__Secure-access-token"
	refreshTokenKey = "__Secure-refresh-token"
)

type (
	contextKey struct {
		name string
	}
)

// private vars
var (
	playerKey = &contextKey{"player"}
)

// Errors
var (
	ErrUnauthenticated    = errs.B().Code(errs.Unauthenticated).Msg("user is unauthenticated").Err()
	ErrSessionInvalidated = errs.B().Code(errs.Unauthenticated).Msg("session is invalid. Please login and try again.").Err()
)

func Player(ctx context.Context) *game.Player {
	v, _ := ctx.Value(playerKey).(*game.Player)
	return v
}

func (h *Handler) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// validate access cookie
		accessCk, err := r.Cookie(accessTokenKey)
		if err == nil {
			err = accessCk.Valid()
		}

		// validate token inside access cookie
		var player game.Player
		if err == nil {
			player, err = h.token.Validate(ctx, auth.Token(accessCk.Value))
		}

		isDBPlayer := false
		if err != nil {
			if player, accessCk, err = h.refreshCookie(ctx, w, r); err != nil {
				resp.Error(w, err)
				return
			}
			isDBPlayer = true
		}

		// during refresh, player is already updated. This check is needed to
		// update player if token is not refreshed.
		if !isDBPlayer {
			if err = h.dbPlayerFromToken(ctx, &player); err != nil {
				resp.Error(w, err)
				return
			}
		}

		// replace the request context
		ctx = context.WithValue(ctx, playerKey, &player)
		r = r.WithContext(ctx)

		// pass to the next handler
		next.ServeHTTP(w, r)
	})
}

// refreshCookie regenerates accessToken based on refreshToken. RefreshCookie is annuled if any error occurs with the refreshToken itself,
// thereby triggering client reauthentication
func (h *Handler) refreshCookie(ctx context.Context, w http.ResponseWriter, r *http.Request) (p game.Player, c *http.Cookie, err error) {
	refreshCk, err := r.Cookie(refreshTokenKey)
	if err != nil {
		return p, nil, ErrUnauthenticated
	}

	defer func() {
		if err == nil {
			return
		}
		var detailed *errs.Error
		errors.As(errs.Convert(err), &detailed)
		if detailed.Code == errs.Unauthenticated {
			deleteCookie(w, refreshCk)
		}
	}()
	if err = refreshCk.Valid(); err != nil {
		return p, nil, errs.B().Code(errs.Unauthenticated).Msg(err.Error()).Err()
	}

	player, err := h.token.Validate(ctx, auth.Token(refreshCk.Value))
	if err != nil {
		return p, nil, ErrUnauthenticated
	}
	if err = h.dbPlayerFromToken(ctx, &player); err != nil {
		return p, nil, err
	}
	accessToken, err := h.token.Generate(ctx, player, accessTokenTTL)
	if err != nil {
		return p, nil, ErrUnauthenticated
	}
	ck := newAccessCookie(accessToken)
	return player, &ck, nil
}

// playerFromToken fetches the player, validates, and updates the pointer passed
func (h *Handler) dbPlayerFromToken(ctx context.Context, tp *game.Player) error {
	if tp == nil {
		return errs.B().Code(errs.Internal).Msg("fatal: nil player").Err()
	}
	dbPlayer, err := h.srv.GetPlayer(ctx, tp.Username)
	if err != nil {
		return err
	}
	if dbPlayer.SessionTs != tp.SessionTs {
		return ErrSessionInvalidated
	}
	*tp = *dbPlayer
	return nil
}

func decodeHeader(auth string) (string, error) {
	spl := strings.Split(auth, " ")
	if len(spl) == 2 {
		if strings.ToLower(spl[0]) != "bearer" {
			return "", ErrUnauthenticated
		}
		return spl[1], nil
	}
	return "", ErrUnauthenticated
}
