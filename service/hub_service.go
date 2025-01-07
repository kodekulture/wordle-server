package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kodekulture/wordle-server/game"
)

const (
	// RoomDuration is the maximum time for a game, room is deleted and game data is wiped if
	// game is not played during this time.
	RoomDuration = time.Hour
	// EmptyRoomDuration is the maximum time a game is left without players. It is shorter than RoomDuration
	// because Room is probably not in use anymore and contains no data.
	EmptyRoomDuration = time.Minute * 15
)

type hub struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]*game.Room
}

// NewStorage returns a new Storage.
func newHub(ctx context.Context, r map[uuid.UUID]*game.Room) *hub {
	if r == nil {
		r = make(map[uuid.UUID]*game.Room)
	}
	h := hub{rooms: r}
	go h.gc(ctx)

	return &h
}

// GetRoom returns the room with the given id and a bool indicating whether the room was found.
func (s *hub) GetRoom(id uuid.UUID) (*game.Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rooms[id]
	return r, ok
}

// SetRoom sets the room with the given id to the given room.
func (s *hub) SetRoom(id uuid.UUID, r *game.Room) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[id] = r
}

// DeleteRoom deletes the room with the given id.
func (s *hub) DeleteRoom(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rooms, id)
}

func (s *hub) gc(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	isMarkPhase := true
	garbage := make([]*game.Room, 0)

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
