package handler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/Chat-Map/wordle-server/game/word"
	"github.com/Chat-Map/wordle-server/service"
)

var Hub struct {
	rooms map[uuid.UUID]*Room
	mu    sync.RWMutex // protects the room map
	s     *service.Service
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
	CFinish Event = "client/finish"

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
	// This context is used to protect writes in room's closed channels
	// When sending to any of the room's channels(leaveChan, broadcast) this context
	// must be still active, Otherwise the sending should not be initiated.
	// As a tip use a select stm when sending messsage to any of the room's channels
	ctx       context.Context
	cancelCtx func() // cancel the room's context

	mu        sync.Mutex // protects players map
	players   map[string]*PlayerConn
	broadcast chan Payload
	leaveChan chan *PlayerConn // leftChan is used to notify the room when a player leaves
	g         *game.Game

	active bool // whether the game has started
	closed bool // whether the game has finished
}

// NewRoom creates a new room and add it to the Hub.
func NewRoom(game *game.Game) *Room {
	ctx, cancel := context.WithCancel(context.Background())
	room := Room{
		ctx:       ctx,
		cancelCtx: cancel,
		players:   make(map[string]*PlayerConn),
		broadcast: make(chan Payload),
		leaveChan: make(chan *PlayerConn),
		g:         game,
	}

	Hub.mu.Lock()
	Hub.rooms[room.g.ID] = &room
	Hub.mu.Unlock()
	go room.run()
	go room.leave()
	return &room
}

// start Process `SStart` event and broadcasts a `CStart` event to all players in the room.
func (r *Room) start(m Payload) {
	// Check if the player is the creator of the game
	if r.g.Creator != m.From {
		m.sender.write(newPayload(CError, "Only the game's creator can start the game", ""))
		return
	}
	// Check if the game has already started
	if r.active {
		m.sender.write(newPayload(CError, "Game already started", ""))
		return
	}
	r.g.Start()
	// Save the game to the database
	// TODO: Uncomment this when the database is ready
	_ = Hub.s.StartGame(r.ctx, r.g)
	// if err != nil {
	// 	m.sender.write(newPayload(CError, "Failed to start game", ""))
	// 	return
	// }
	r.active = true
	r.sendAll(newPayload(CStart, "Game started!", ""))
}

// message process `SMessage` event and broadcasts a `CMessage` event to all players in the room.
func (r *Room) message(m Payload) {
	text, ok := m.Data.(string)
	if !ok {
		m.sender.write(newPayload(CError, "Invalid message type", ""))
		return
	}
	r.sendAll(newPayload(CMessage, text, m.From))
}

// play Process `SPlay` event and broadcasts a `CPlay` event to all players in the room
// and `CResult` event to the player who submitted the message.
func (r *Room) play(m Payload) {
	// If the game has not started, return an error
	if !r.active {
		m.sender.write(newPayload(CError, "Room isn't active", ""))
		return
	}
	session := r.g.Sessions[m.sender.Username]
	// If the user is not in the game, return an error
	if session == nil {
		m.sender.write(newPayload(CError, "Invalid user session", ""))
		return
	}
	// Check if the user already won
	if session.Won() {
		m.sender.write(newPayload(CError, "You already won", ""))
		return
	}
	// Check if the user already used all their attempts or won
	if !session.CanPlay() {
		m.sender.write(newPayload(CError, "You already used all your attempts", ""))
		return
	}
	// Parse message and send error if type is not string
	text, ok := m.Data.(string)
	if !ok {
		m.sender.write(newPayload(CError, "Invalid message", ""))
		return
	}
	// Check given word length
	if len(text) != word.Length {
		m.sender.write(newPayload(CError, "Invalid message string length", ""))
		return
	}
	// Process the given word and send error if the word is invalid
	w := word.New(text)
	ok = r.g.Play(m.sender.PName(), &w)
	if !ok {
		m.sender.write(newPayload(CError, "Invalid word", ""))
		return
	}

	// Send the result to the player who submitted the message
	m.sender.write(newPayload(CResult, w.Stats, ""))

	// Send the result to all players in the room
	text = fmt.Sprintf("%s got %d/%d correct", m.From, w.CorrectCount(), len(w.Word))
	r.sendAll(newPayload(CPlay, text, m.From))

	// Check if the game has finished, if so, close the room
	if r.g.HasEnded() {
		r.sendAll(newPayload(CFinish, "Game has ended", ""))
		r.close()
		err := Hub.s.FinishGame(context.Background(), r.g)
		if err != nil {
			log.Printf("failed to finish game: %v", err)
		}
	}
}

func (r *Room) join(user game.Player, conn *websocket.Conn) {
	r.mu.Lock()
	old := r.players[user.Username]
	r.mu.Unlock()
	// Kickout the old player with the same username
	if old != nil {
		r.kickout(old)
	}
	// Create a new playerConn
	new := newPlayerConn(conn, r, user.Username)
	// Create a new session for the user if it doesn't exist
	if _, ok := r.g.Sessions[user.Username]; !ok {
		r.g.Join(user)
	}
	// Add the `new` player to the room and remove the `old` player
	r.mu.Lock()
	r.players[user.Username] = new
	r.mu.Unlock()
	// Send the player his current state in the game
	new.write(newPayload(CData, r.g.Sessions[user.Username].Guesses, ""))
	// Notify players that that a new player has joined
	text := fmt.Sprintf("%s has joined", new.PName())
	r.sendAll(newPayload(CJoin, text, ""))
}

// kickout kicks out a player from the room.
// Sends a username to the `leaveChan` channel or do nothing
// if the room is closed(i.e. context cancelled).
func (r *Room) kickout(p *PlayerConn) {
	select {
	case <-r.ctx.Done():
	case r.leaveChan <- p:
	}
}

// leave broadcasts a `CLeave` event to all players in the room.
func (r *Room) leave() {
	for p := range r.leaveChan {
		r.mu.Lock()
		p.Close()
		p.room = nil // set room to nil to free memory
		// If currect loged in user is the same as the player
		// that is being kicked out, remove the player from the room.
		// This check is made to avoid kicking a the new player who just joined
		if r.players[p.Username] == p {
			delete(r.players, p.PName())
		}
		r.mu.Unlock()
		r.sendAll(newPayload(CLeave, fmt.Sprintf("%s has left", p.PName()), ""))
	}
}

// close closes the room and all players in the room.
// This is used when the game is finished.
func (r *Room) close() {
	if r.closed {
		return
	}
	r.closed = true
	r.active = false
	// Cancel the context to stop the `leave` goroutine and close
	// all prevent any new players from sending messages to the room.
	r.cancelCtx()
	r.mu.Lock()
	// Close all players connection
	for _, p := range r.players {
		p.Close()
		delete(r.players, p.PName())
	}
	r.mu.Unlock()
	close(r.broadcast)
	close(r.leaveChan)
	// Remove the room from the hub to free memory
	Hub.mu.Lock()
	delete(Hub.rooms, r.g.ID)
	Hub.mu.Unlock()
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
			message.sender.write(newPayload(CError, "Unknown message type", ""))
		}
	}
}

// sendAll sends the payload to all players in the room.
// If sending the payload fails, the player is removed from the room.
func (r *Room) sendAll(payload Payload) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.players {
		err := p.write(payload)
		if err != nil {
			// r.kickout(p) is called in a goroutine to avoid mutext lock
			// since the lock is already acquired in this function
			go func(p *PlayerConn) {
				r.kickout(p)
			}(p)
		}
	}
}

var (
	// pongWait is how long we will await a pong response from player
	pongWait = 10 * time.Second

	pingInterval = (pongWait * 9) / 10
)

// PlayerConn represents a player in the game.
// A player can be in multiple rooms, but only one game at a time.
type PlayerConn struct {
	conn     *websocket.Conn
	room     *Room
	Username string
	writeMu  sync.Mutex

	t *time.Ticker
}

// PName returns the player name.
func (p *PlayerConn) PName() string {
	return p.Username
}

// newPlayerConn creates a new player.
// This function starts the read goroutine to forward messages to the room.
// Also starts the ping goroutine to ping the player every 5 seconds
// to check if the player is still connected otherwise the connection is closed.
func newPlayerConn(conn *websocket.Conn, room *Room, username string) *PlayerConn {
	// Create a ticker to ping the player every 5 seconds
	// The ticker is stored in the player struct so that it can be stopped
	// on the player.Close() call.
	ticker := time.NewTicker(time.Second * 5)
	player := PlayerConn{
		Username: username,
		conn:     conn,
		room:     room,
		t:        ticker,
	}
	go player.read()
	go player.ping()
	return &player
}

// Close closes the player connection.
func (p *PlayerConn) Close() error {
	p.t.Stop()
	return p.conn.Close()
}

// ping pings the player every 5 seconds to check if the player is still connected
// otherwise the connection is closed.
func (p *PlayerConn) ping() {
	defer p.t.Stop()
	for range p.t.C {
		p.writeMu.Lock()
		err := p.conn.WriteMessage(websocket.PingMessage, []byte{})
		p.writeMu.Unlock()
		if err != nil {
			p.room.kickout(p)
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
			// If the error is not a close error, then the player is kicked out.
			// if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
			// 	p.room.kickout(p)
			// }
			return
		}
		payload.From = p.Username // From set by the client is ignored by the server for security reasons.
		payload.sender = p
		select {
		case <-p.room.ctx.Done():
			return
		case p.room.broadcast <- payload:
		}
	}
}

// write writes the payload to the player connection in synchronized manner.
func (p *PlayerConn) write(payload Payload) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	err := p.conn.WriteJSON(payload)
	if err != nil {
		log.Printf("Error writing to player (%s): %s", p.PName(), err)
	}
	return err
}
