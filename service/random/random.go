package random

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

var (
	defaultLenght = 30
	valueMaxLife  = time.Hour
	cleanupCycle  = time.Minute
	characters    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type value struct {
	createdAt time.Time
	username  string
	gameID    uuid.UUID
}

type RandomGen struct {
	r *rand.Rand
	s map[string]value
	l int
}

// New returns a new RandomGen
func New(ctx context.Context) RandomGen {
	r := RandomGen{
		r: rand.New(rand.NewSource(time.Now().UnixNano())),
		l: defaultLenght,
		s: make(map[string]value),
	}
	go r.cleanup(ctx)
	return r
}

// Store stores the username and gameID associated with a the token and returns it
func (rg RandomGen) Store(username string, gameID uuid.UUID) string {
	token := make([]byte, rg.l)
	for i := 0; i < rg.l; i++ {
		token[i] = characters[rg.r.Intn(len(characters))]
	}
	tokenString := string(token)
	rg.s[tokenString] = value{
		username:  username,
		gameID:    gameID,
		createdAt: time.Now(),
	}
	return tokenString
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
