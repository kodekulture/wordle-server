package handler

import (
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	// Create upgrade websocket connection
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024 * 1024,
		WriteBufferSize: 1024 * 1024,
		//Solving cross-domain problems
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func (h *Handler) live(w http.ResponseWriter, r *http.Request) {
	// Parse username from request query
	username := r.URL.Query().Get("username")
	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid username"))
	}

	// Parse roomID from request query
	roomID, err := uuid.Parse(r.URL.Query().Get("room_id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid room id"))
		return
	}

	// Get room from Hub
	Hub.mu.Lock()
	defer Hub.mu.Unlock()
	room, ok := Hub.rooms[roomID]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("game not found"))
		return
	}

	// Check the game has not been closed
	if room.closed {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("game has been closed"))
		return
	}

	// Check if the game has started already
	if room.active && room.g.Sessions[username] == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("you can't join ongoing game"))
		return
	}

	// Upgrade the HTTP connection to a websocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading connection: %v", err)
		return
	}

	go room.join(username, conn)
}
