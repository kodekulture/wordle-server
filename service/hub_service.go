package service

import (
	"sync"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/google/uuid"
)

type hub struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]*game.Room
}

// NewStorage returns a new Storage.
func newHub(r map[uuid.UUID]*game.Room) *hub {
	if r == nil {
		r = make(map[uuid.UUID]*game.Room)
	}
	return &hub{rooms: r}
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
