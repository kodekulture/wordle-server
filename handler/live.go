package handler

import (
	"log"
	"net/http"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/gorilla/websocket"
	"github.com/lordvidex/errs"
	"github.com/lordvidex/x/resp"
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
	// Parse token from request query
	token := r.URL.Query().Get("token")
	username, gameID, ok := h.srv.GetTokenPayload(token)
	if !ok {
		resp.Error(w, errs.B().Code(errs.InvalidArgument).Msg("invalid token").Err())
		return
	}

	// Get room from Hub
	Hub.mu.Lock()
	room, ok := Hub.rooms[gameID]
	Hub.mu.Unlock()
	if !ok {
		resp.Error(w, errs.B().Code(errs.InvalidArgument).Msg("game not found").Err())
		return
	}

	// Check the game has not been closed
	if room.closed {
		resp.Error(w, errs.B().Code(errs.InvalidArgument).Msg("game has been closed").Err())
		return
	}

	// Check if the game has started already
	if room.active && room.g.Sessions[username] == nil {
		resp.Error(w, errs.B().Code(errs.InvalidArgument).Msg("you can't join ongoing game").Err())
		return
	}

	var (
		player game.Player
	)
	// Check if the user already has a session
	sess := room.g.Sessions[username]
	if sess == nil {
		// Fetch the player from the database
		pl, err := h.srv.GetPlayer(r.Context(), username)
		if err != nil {
			resp.Error(w, err)
			return
		}
		player = *pl
	} else {
		player = sess.Player
	}

	// Upgrade the HTTP connection to a websocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading connection: %v", err)
		return
	}

	go room.join(player, conn)
}
