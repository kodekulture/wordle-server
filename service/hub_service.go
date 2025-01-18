package service

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kodekulture/wordle-server/game"
	"github.com/kodekulture/wordle-server/internal/config"
)

const (
	// RoomDuration is the maximum time for a game, room is deleted and game data is wiped if
	// game is not played during this time.
	RoomDuration = time.Hour
	// EmptyRoomDuration is the maximum time a game is left without players. It is shorter than RoomDuration
	// because Room is probably not in use anymore and contains no data.
	EmptyRoomDuration = time.Minute * 15
)

type localStorage struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]*game.Room
}

// NewStorage returns a new Storage.
func newLocalStorage(ctx context.Context) *localStorage {
	h := localStorage{rooms: make(map[uuid.UUID]*game.Room)}
	go h.gc(ctx)

	return &h
}

// GetRoom returns the room with the given id and a bool indicating whether the room was found.
func (s *localStorage) GetRoom(id uuid.UUID) (*game.Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rooms[id]
	return r, ok
}

// SetRoom sets the room with the given id to the given room.
func (s *localStorage) SetRoom(id uuid.UUID, r *game.Room) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[id] = r
}

// DeleteRoom deletes the room with the given id.
func (s *localStorage) DeleteRoom(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rooms, id)
}

func (s *localStorage) gc(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	isMarkPhase := true
	garbage := make([]*game.Room, 0)

	enabled := config.GetOrDefault("GC", true, strconv.ParseBool)

	if !enabled {
		return
	}

	mark := func() {
		garbage = nil
		s.mu.RLock()
		defer s.mu.RUnlock()

		for _, r := range s.rooms {
			g := r.Game()
			if g == nil {
				garbage = append(garbage, r)
				continue
			}

			if time.Since(g.CreatedAt) >= RoomDuration {
				garbage = append(garbage, r)
				continue
			}

			if time.Since(g.CreatedAt) >= EmptyRoomDuration && len(g.Sessions) == 0 {
				garbage = append(garbage, r)
				continue
			}
		}
	}

	sweep := func() {
		// 1. fast remove from rooms
		s.mu.Lock()
		for _, r := range garbage {
			v, _ := uuid.Parse(r.ID())
			delete(s.rooms, v)
		}
		s.mu.Unlock()

		// 2. cleanup later
		for _, r := range garbage {
			if r.IsClosed() {
				continue
			}
			r.Close()
		}
		garbage = nil
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if isMarkPhase {
				mark()
			} else {
				// sweep phase
				sweep()
			}

			isMarkPhase = !isMarkPhase
		}
	}
}
