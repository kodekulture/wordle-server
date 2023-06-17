package random

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

var (
	valueMaxLife = time.Hour
	cleanupCycle = time.Minute
	salt         = "NdZXxlv1ShnypBDGrJCRe8g7HPENVyXkSZyOsSYyQbGtqnduoxMPyfcnKXEVKdHz" // TODO: load from env
)

type value struct {
	createdAt time.Time
	username  string
	gameID    uuid.UUID
}

type RandomGen struct {
	s map[string]value
}

// New returns a new RandomGen
func New(ctx context.Context) RandomGen {
	r := RandomGen{s: make(map[string]value)}
	go r.cleanup(ctx)
	return r
}

// Store stores the username and gameID associated with a the token and returns it
func (rg RandomGen) Store(username string, gameID uuid.UUID) string {
	hash256 := sha256.New()
	data := username + salt + gameID.String()
	hash256.Write([]byte(data))
	token := hex.EncodeToString(hash256.Sum(nil))
	// if the user already issued the token
	if _, ok := rg.s[token]; ok {
		return token
	}
	rg.s[token] = value{
		username:  username,
		gameID:    gameID,
		createdAt: time.Now(),
	}
	return token
}

// Get returns the username and gameID associated with the token
func (r RandomGen) Get(token string) (string, uuid.UUID, bool) {
	v, ok := r.s[token]
	if !ok {
		return "", uuid.Nil, false
	}
	return v.username, v.gameID, true
}

// cleanup removes all the values that are older than valueMaxLife
// since the game is not supposed to last more than one hour
func (r RandomGen) cleanup(ctx context.Context) {
	ticker := time.NewTicker(cleanupCycle)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			for k, v := range r.s {
				if now.Sub(v.createdAt) > valueMaxLife {
					delete(r.s, k)
				}
			}
		}
	}

}
