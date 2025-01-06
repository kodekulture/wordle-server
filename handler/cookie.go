package handler

import (
	"net/http"
	"time"

	"github.com/kodekulture/wordle-server/internal/config"
	"github.com/lordvidex/x/auth"
)

func isProd() bool {
	return config.Get("ENV") == "prod"
}

const (
	accessTokenTTL  = 1 * time.Hour        // 1 hr
	refreshTokenTTL = 365 * 24 * time.Hour // 1 year
)

func newAccessCookie(token auth.Token) http.Cookie {
	return http.Cookie{
		Name:     accessTokenKey,
		Value:    string(token),
		Expires:  time.Now().Add(accessTokenTTL),
		Secure:   isProd(), // enable development usage
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}
}

func newRefreshCookie(token auth.Token) http.Cookie {
	return http.Cookie{
		Name:     refreshTokenKey,
		Value:    string(token),
		Expires:  time.Now().Add(refreshTokenTTL),
		Secure:   isProd(), // enable development usage
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func deleteCookie(w http.ResponseWriter, c *http.Cookie) {
	c.MaxAge = -1
	c.Expires = time.Unix(0, 0)
	http.SetCookie(w, c)
}
