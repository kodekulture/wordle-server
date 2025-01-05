package game

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lordvidex/x/ptr"
	"github.com/rs/zerolog/log"

	"github.com/kodekulture/wordle-server/game/word"
)

type Event string

const (
	SMessage Event = "server/message"
	CMessage Event = "client/message"

	SPlay   Event = "server/play"
	CPlay   Event = "client/play"
	CFinish Event = "client/finish"

	SStart Event = "server/start"
	CStart Event = "client/start"

	CJoin  Event = "client/join"
	CLeave Event = "client/leave"

	CData  Event = "client/data"
	CError Event = "client/error"

	PJoin       Event = "private/join"
	PLeave      Event = "private/leave"
	PKickout    Event = "private/kickout"
	PDisconnect Event = "private/disconnect"
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

type GameSaver interface {
	FinishGame(context.Context, *Game) error
	StartGame(context.Context, *Game) error
}

type Room struct {
	// This context is used to protect writes in room's closed channels
	// When sending to any of the room's channels(leaveChan, broadcast) this context
	// must be still active, Otherwise the sending should not be initiated.
	// As a tip use a select stm when sending messsage to any of the room's channels
	ctx       context.Context
	cancelCtx func() // cancel the room's context

	players   map[string]*PlayerConn
	broadcast chan Payload
	g         *Game

	active bool // whether the game has started
	closed bool // whether the game has finished

	saver GameSaver
}

// ID returns the ID of the room which is the ID of the game
func (r *Room) ID() string {
	return r.g.ID.String()
}

// Game returns the game of the room
func (r *Room) Game() *Game {
	return r.g
}

// Join adds a player to the room
func (r *Room) Join(p Player, conn *websocket.Conn) {
	pc := newPlayerConn(conn, r, p)
	r.tryBroadcast(newPayload(PJoin, pc, ""))
}

// CanJoin checks if a player can join the room
func (r *Room) CanJoin(username string) error {
	if r.IsClosed() {
		return errors.New("the room is closed")
	}
	_, ok := r.g.Sessions[username]
	if r.active && !ok {
		return errors.New("the game has already started")
	}
	return nil
}

// IsClosed checks if the room is closed
func (r *Room) IsClosed() bool {
	return r.closed
}

// NewRoom creates a new room and add it to the Hub.
func NewRoom(game *Game, storer GameSaver) *Room {
	ctx, cancel := context.WithCancel(context.Background())
	room := &Room{
		ctx:       ctx,
		cancelCtx: cancel,
		players:   make(map[string]*PlayerConn),
		broadcast: make(chan Payload),
		g:         game,
		saver:     storer,

		active: game.StartedAt != nil && game.EndedAt == nil,
		closed: game.EndedAt != nil,
	}
	go room.run()
	return room
}

// start Process `SStart` event and broadcasts a `CStart` event to all players in the room.
func (r *Room) start(m Payload) {
	pconn := m.sender
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
	if r.saver != nil {
		err := r.saver.StartGame(r.ctx, r.g)
		if err != nil {
			m.sender.write(newPayload(CError, "Failed to start game", ""))
			return
		}
	}
	r.active = true
	r.sendAll(newPayload(CStart, "Game started!", ""))
	r.sendAll(newPayload(CData, ToInitialData(ptr.ToObj(r.g), pconn.PName()), ""))
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

var letterRegexp = regexp.MustCompile("^[a-zA-Z]+$")

// play Process `SPlay` event and broadcasts a `CPlay` event to all players in the room
// and `CResult` event to the player who submitted the message.
func (r *Room) play(m Payload) {
	// If the game has not started, return an error
	if !r.active {
		m.sender.write(newPayload(CError, "Room isn't active", ""))
		return
	}
	session := r.g.Sessions[m.sender.PName()]
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
	// Check if the given word is valid
	if !letterRegexp.MatchString(text) {
		m.sender.write(newPayload(CError, "Invalid message characters", ""))
		return
	}
	// Process the given word and send error if the word is invalid
	w := word.New(text)
	dRank, err := r.g.Play(m.sender.PName(), &w)
	if err != nil {
		m.sender.write(newPayload(CError, err.Error(), ""))
		return
	}

	// Send the result to all players in the room
	result := PlayerGuessResponse{
		Result:      ToGuess(w, false),
		RankOffset:  ptr.Obj(dRank),
		Leaderboard: ToLeaderboard(r.g.Leaderboard),
	}
	r.sendAll(newPayload(CPlay, result, m.From))

	// Check if the game has finished, if so, close the room
	if r.g.HasEnded() {
		r.sendAll(newPayload(CFinish, "Game has ended", ""))
		r.close()
	}
}

func (r *Room) join(m Payload) {
	pconn := m.Data.(*PlayerConn)
	old := r.players[pconn.PName()]
	// If the player is already in the room, kick him out.
	if old != nil {
		r.leave(newPayload(PKickout, old, ""))
	}
	// Create a new session for the user if it doesn't exist.
	if _, ok := r.g.Sessions[pconn.PName()]; !ok {
		r.g.Join(pconn.player)
	}
	// Send the player his current state in the game.
	// On error, close the player connection since he will have inconsistent data with which he can't play the game.
	err := pconn.write(newPayload(CData, ToInitialData(ptr.ToObj(r.g), pconn.PName()), ""))
	if err != nil {
		log.Err(err).Caller().Msg("failed to send player data")
		err = pconn.close()
		if err != nil {
			log.Err(err).Caller().Msg("failed to close player connection")
		}
		return
	}
	r.players[pconn.PName()] = pconn
	r.sendAll(newPayload(CJoin, fmt.Sprintf("%s has joined", pconn.PName()), pconn.PName()))
}

// leave process `SLeave` and `SKickout` events and broadcasts a `CLeave` event to all players in the room.
// Also the player connection is closed and  is removed from the room (if the current user is the same as the player being kicked out).
func (r *Room) leave(m Payload) {
	var players []*PlayerConn
	switch m.Data.(type) {
	case *PlayerConn:
		players = append(players, m.Data.(*PlayerConn))
	case []*PlayerConn:
		players = m.Data.([]*PlayerConn)
	default:
		log.Info().Msgf("Unkown payload type provided for leave: %#v", m.Data)
		return
	}
	// Process the player list and close their connections
	for i, p := range players {
		// If the current player isn't the one in player room do nothing, since this function is called by many others
		// like `sendAll`, `plauerConn.write` and `playerConn.read` so we might get repeated requests having the same `playerConn`
		// Notice that the `players[i]` is set to `nil` to avoid sending message twice for kicking out the signle users
		if !p.active {
			players[i] = nil
			continue
		}
		p.close()
		delete(r.players, p.PName())
	}
	for _, p := range players {
		// `p` is nil if and only if the player has already been kicked out
		if p == nil {
			continue
		}
		var text string
		if m.Type == PKickout {
			text = fmt.Sprintf("%s has been kicked out", p.PName())
		} else { // PLeave
			text = fmt.Sprintf("%s has left", p.PName())
		}
		r.sendAll(newPayload(CLeave, text, ""))
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
	// Close all players connection
	for _, p := range r.players {
		p.close()
		delete(r.players, p.PName())
	}
	close(r.broadcast)
	// Store the game in the database
	if r.saver != nil {
		err := r.saver.FinishGame(context.Background(), r.g)
		if err != nil {
			log.Err(err).Caller().Msg("failed to store game")
		}
	}
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
		case PJoin:
			r.join(message)
		case PLeave:
			r.leave(message)
		default:
			message.sender.write(newPayload(CError, "Unknown message type", ""))
		}
	}
}

// tryBroadcast tries to broadcast the payload to all players in the room if the room is active.
func (r *Room) tryBroadcast(payload Payload) {
	select {
	case <-r.ctx.Done():
	case r.broadcast <- payload:
	}
}

// sendAll sends the payload to all players in the room.
// If sending the payload fails, the player is removed from the room.
func (r *Room) sendAll(payload Payload) {
	errs := make([]*PlayerConn, 0)
	for _, p := range r.players {
		err := p.write(payload)
		if err != nil {
			errs = append(errs, p)
		}
	}
	if len(errs) != 0 {
		r.leave(newPayload(PLeave, errs, ""))
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
	conn    *websocket.Conn
	room    *Room
	player  Player
	writeMu sync.Mutex
	active  bool // indicator for player's connection status

	t *time.Ticker
}

// PName returns the player name.
func (p *PlayerConn) PName() string {
	return p.player.Username
}

// newPlayerConn creates a new player.
// This function starts the read goroutine to forward messages to the room.
// Also starts the ping goroutine to ping the player every 5 seconds
// to check if the player is still connected otherwise the connection is closed.
func newPlayerConn(conn *websocket.Conn, room *Room, player Player) *PlayerConn {
	// Create a ticker to ping the player every 5 seconds
	// The ticker is stored in the player struct so that it can be stopped
	// on the player.Close() call.
	ticker := time.NewTicker(pingInterval)
	p := PlayerConn{
		player: player,
		conn:   conn,
		room:   room,
		active: true,

		t: ticker,
	}
	go p.read()
	go p.ping()
	return &p
}

// Close closes the player connection.
func (p *PlayerConn) close() error {
	p.active = false
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
			p.room.tryBroadcast(newPayload(PLeave, p, ""))
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
			p.room.tryBroadcast(newPayload(PLeave, p, ""))
			break
		}
		// if the payload type is not prefixed with "server/" then it is not allowed to be sent by the player.
		if !strings.HasPrefix(string(payload.Type), "server/") {
			p.write(newPayload(CError, "unsupported action", ""))
			continue
		}
		payload.From = p.PName() // From set by the client is ignored by the server for security reasons.
		payload.sender = p
		p.room.tryBroadcast(payload)
	}
}

// write writes the payload to the player connection in synchronized manner.
func (p *PlayerConn) write(payload Payload) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	err := p.conn.WriteJSON(payload)
	if err != nil {
		log.Err(err).Caller().Msgf("Error writing to player (%s)", p.PName())
	}
	return err
}
