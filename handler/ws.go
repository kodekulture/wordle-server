package handler

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/Chat-Map/wordle-server/game/word"
)

var Hub struct {
	rooms map[uuid.UUID]*Room
	mu    sync.Mutex // protects the room map
}

func init() {
	Hub.rooms = make(map[uuid.UUID]*Room)
}

type Event string

const (
	SMessage Event = "server/message"
	CMessage Event = "client/message"

	SPlay   Event = "server/play"
	CResult Event = "client/result"
	CPlay   Event = "client/play"

	SStart Event = "server/start"
	CStart Event = "client/start"

	CJoin  Event = "client/join"
	CLeave Event = "client/leave"

	CData  Event = "client/data"
	CError Event = "client/error"
)

type Payload struct {
	Type   Event       `json:"event"`
	Data   interface{} `json:"data"`
	From   string      `json:"from"` // From is the name of the player that sent the message displayed to all other players in the room
	sender *PlayerConn `json:"-"`    // sender is the player that sent the message
}

func newPayload(event Event, data interface{}, from string) Payload {
	return Payload{
		Type: event,
		Data: data,
		From: from,
	}
}

type Room struct {
	mu        sync.Mutex // protects players map
	players   map[string]*PlayerConn
	broadcast chan Payload
	g         *game.Game

	active bool // whether the game has started
	closed bool // whether the game has finished
}

// NewRoom creates a new room and add it to the Hub.
func NewRoom(game *game.Game) *Room {
	room := Room{
		players:   make(map[string]*PlayerConn),
		broadcast: make(chan Payload),
		g:         game,
	}

	Hub.mu.Lock()
	Hub.rooms[room.g.ID] = &room
	Hub.mu.Unlock()
	go room.run()
	return &room
}

// start Process `SStart` event and broadcasts a `CStart` event to all players in the room.
func (r *Room) start(message Payload) {
	// Check if the game has already started
	if r.active {
		message.sender.write(newPayload(CError, "Game already started", ""))
		return
	}
	// Check if the player is the creator of the game
	if r.g.Creator != message.From {
		message.sender.write(newPayload(CError, "Only the game's creator can start the game", ""))
		return
	}
	// Update room & game status
	r.active = true
	r.g.Start()
	r.sendAll(newPayload(CStart, "Game started!", ""))
}

// message process `SMessage` event and broadcasts a `CMessage` event to all players in the room.
func (r *Room) message(message Payload) {
	// Parse message and send error if type is not string
	text, ok := message.Data.(string)
	if !ok {
		message.sender.write(newPayload(CError, "Invalid message type", ""))
		return
	}
	// Send the message to all players in the room
	payload := newPayload(CMessage, text, message.From)
	r.sendAll(payload)
}

// play Process `SPlay` event and broadcasts a `CPlay` event to all players in the room
// and `CResult` event to the player who submitted the message.
func (r *Room) play(message Payload) {
	// If the game has not started, return an error
	if !r.active {
		message.sender.write(newPayload(CError, "Room isn't active", ""))
		return
	}
	// Parse message and send error if type is not string
	text, ok := message.Data.(string)
	if !ok {
		message.sender.write(newPayload(CError, "Invalid message", ""))
		return
	}
	// Check given word length
	if len(text) != word.Length {
		message.sender.write(newPayload(CError, "Invalid message string length", ""))
		return
	}
	// Process the given word and send error if the word is invalid
	w := word.New(text)
	sender := message.sender.Username
	ok = r.g.Play(sender, &w)
	if !ok {
		// TODO: what if the user session has ended because ok will be false as well
		message.sender.write(newPayload(CError, "Invalid word", ""))
		return
	}

	// Send the result to the player who submitted the message
	fmt.Println(w.Stats)
	payload := newPayload(CResult, w.Stats, "")
	message.sender.write(payload)

	// Send the result to all players in the room
	// TODO: Create more reasonable message.. we shouldn't also show the word the other user played
	text = fmt.Sprintf("%s played %s", message.From, text)
	payload = newPayload(CPlay, text, message.From)
	r.sendAll(payload)

	// Check if the game has finished, if so, close the room
	if r.g.HasEnded() {
		r.sendAll(newPayload(CMessage, "Game has ended", ""))
		r.close()
	}
}

func (r *Room) Join(username string, conn *websocket.Conn) {
	old := r.players[username]
	if old != nil {
		old.Close()
		// If the player's room is not closed, notify all players in the room
		// that the player has left the room
		if !old.room.closed {
			text := fmt.Sprintf("%s has left", old.PName())
			old.room.sendAll(newPayload(CLeave, text, ""))
		}
		// Close the `old` player connection
		old.room = nil
	}

	// Create a new player and add it to the game
	new := NewPlayerConn(conn, r, username)
	if _, ok := r.g.Sessions[username]; !ok {
		r.g.Join(username)
	}

	// Add the `new` player to the room and remove the `old` player
	r.mu.Lock()
	r.players[username] = new
	r.mu.Unlock()

	// Send the player his current state in the game
	new.write(newPayload(CData, r.g.Sessions[username].Guesses, ""))

	// Notify players that that a new player has joined
	text := fmt.Sprintf("%s has joined", new.PName())
	r.sendAll(newPayload(CJoin, text, ""))

}

// close closes the room and all players in the room.
// This is used when the game is finished.
func (r *Room) close() error {
	defer func() {
		if er := recover(); er != nil {
			log.Println("Error recovered", er)
		}
	}()
	if r.closed {
		return nil // TODO: return room already closed error
	}
	r.closed = true
	r.active = false
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.players {
		p.Close()
	}
	close(r.broadcast)
	return nil
}

// run processes all messages sent to the room.
// This function is blocking until the room is closed (r.broadcast is closed)
func (r *Room) run() {
	for message := range r.broadcast {
		switch message.Type {
		case SStart:
			r.start(message)
		case SMessage:
			r.message(message)
		case SPlay:
			r.play(message)
		default:
			log.Println("Unknown message type", message.Type)
			message.sender.write(newPayload(CError, "Unknown message type", ""))
		}
	}
}

// sendAll sends the payload to all players in the room.
// If sending the payload fails, the player is removed from the room.
func (r *Room) sendAll(payload Payload) {
	r.mu.Lock()
	defer r.mu.Unlock()
	errs := make([]*PlayerConn, 0)
	for _, p := range r.players {
		err := p.write(payload)
		if err != nil {
			errs = append(errs, p)
		}
	}
	// Remove players that failed to receive the payload
	for _, p := range errs {
		delete(r.players, p.Username)
		p.Close()
	}
	go func() {
		// Notify users about kicked players
		for _, p := range r.players {
			for _, kp := range errs {
				text := fmt.Sprintf("%s has left", kp.PName())
				p.write(newPayload(CLeave, text, ""))
			}
		}
	}()
}

// PlayerConn represents a player in the game.
// A player can be in multiple rooms, but only one game at a time.
type PlayerConn struct {
	conn     *websocket.Conn
	room     *Room
	Username string
	writeMu  sync.Mutex
}

// PName returns the player name.
func (p *PlayerConn) PName() string {
	return p.Username
}

// NewPlayerConn creates a new player.
// This function starts the read goroutine to forward messages to the room.
// Also starts the ping goroutine to ping the player every 5 seconds
// to check if the player is still connected otherwise the connection is closed.
func NewPlayerConn(conn *websocket.Conn, room *Room, username string) *PlayerConn {
	player := PlayerConn{
		Username: username,
		conn:     conn,
		room:     room,
	}
	go player.read()
	go player.ping()
	return &player
}

// Close closes the player connection.
func (p *PlayerConn) Close() error {
	return p.conn.Close()
}

// ping pings the player every 5 seconds to check if the player is still connected
// otherwise the connection is closed.
func (p *PlayerConn) ping() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for range ticker.C {
		p.writeMu.Lock()
		err := p.conn.WriteMessage(websocket.PingMessage, []byte{})
		p.writeMu.Unlock()
		if err != nil {
			p.Close()
			return
		}
	}
}

// read reads messages from the player connection and forwards
// them to the room to be processed.
func (p *PlayerConn) read() {
	for {
		var payload Payload
		err := p.conn.ReadJSON(&payload)
		if err != nil {
			return
		}
		payload.From = p.Username // From set by the client is ignored by the server for security reasons.
		payload.sender = p
		p.room.broadcast <- payload
	}
}

// write writes the payload to the player connection in synchronized manner.
func (p *PlayerConn) write(payload Payload) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	return p.conn.WriteJSON(payload)
}
