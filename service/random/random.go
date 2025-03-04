package random

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kodekulture/wordle-server/game"
)

var (
	valueMaxLife = time.Hour
	cleanupCycle = time.Minute
	salt         = "NdZXxlv1ShnypBDGrJCRe8g7HPENVyXkSZyOsSYyQbGtqnduoxMPyfcnKXEVKdHz" // TODO: load from env
)

type value struct {
	createdAt time.Time
	player    game.Player
	gameID    uuid.UUID
}

type RandomGen struct {
	mu *sync.RWMutex
	s  map[string]value
}

// New returns a new RandomGen
func New(ctx context.Context) RandomGen {
	r := RandomGen{s: make(map[string]value), mu: new(sync.RWMutex)}
	go r.cleanup(ctx)
	return r
}

// Store stores the username and gameID associated with a the token and returns it
func (rg RandomGen) Store(player game.Player, gameID uuid.UUID) string {
	hash256 := sha256.New()
	data := player.Username + salt + gameID.String()
	hash256.Write([]byte(data))
	token := hex.EncodeToString(hash256.Sum(nil))

	rg.mu.Lock()
	defer rg.mu.Unlock()

	// if the user already issued the token
	if _, ok := rg.s[token]; ok {
		return token
	}
	rg.s[token] = value{
		player:    player,
		gameID:    gameID,
		createdAt: time.Now(),
	}
	return token
}

// Get returns the username and gameID associated with the token
func (r RandomGen) Get(token string) (game.Player, uuid.UUID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.s[token]
	if !ok {
		return game.Player{}, uuid.Nil, false
	}
	return v.player, v.gameID, true
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
			r.mu.Lock()
			for k, v := range r.s {
				if now.Sub(v.createdAt) > valueMaxLife {
					delete(r.s, k)
				}
			}
			r.mu.Unlock()
		}
	}
}
