package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lordvidex/errs"
	"github.com/lordvidex/x/auth"
	"github.com/lordvidex/x/req"
	"github.com/lordvidex/x/resp"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/Chat-Map/wordle-server/game/word"
	"github.com/Chat-Map/wordle-server/handler/token"
	"github.com/Chat-Map/wordle-server/service"
)

type Handler struct {
	s       *http.Server
	router  chi.Router
	srv     *service.Service
	token   token.Handler
	wordGen word.Generator
}

func New(srv *service.Service, tokenHandler token.Handler) *Handler {
	h := &Handler{
		router:  chi.NewRouter(),
		srv:     srv,
		token:   tokenHandler,
		wordGen: word.NewLocalGen(),
	}

	Hub.s = srv
	h.setup()
	return h
}

func (h *Handler) Start(port string) error {
	h.s = &http.Server{Addr: ":" + port, Handler: h.router}
	return h.s.ListenAndServe()
}

func (h *Handler) setup() {
	r := h.router
	// Public routes
	r.Group(func(r chi.Router) {
		r.Get("/health", h.health)
		r.Post("/login", h.login)
		r.Post("/register", h.register)
		r.Get("/live", h.live)
	})

	// Private routes
	r.Group(func(r chi.Router) {
		r.Use(h.authMiddleware(AuthDecodeTypeFetch))

		r.Post("/room", h.createRoom)
		r.Get("/join/room/{id}", h.joinRoom)
		r.Get("/room", h.rooms)
		r.Get("/room/{id}", h.room)
	})

}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

type loginParams struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var payload loginParams
	defer r.Body.Close()
	if err := req.I.Will().Bind(r, &payload).Validate(payload).Err(); err != nil {
		resp.Error(w, err)
		return
	}

	// try finding the user
	var (
		player *game.Player
		err    error
		token  auth.Token
	)
	if player, err = h.srv.GetPlayer(r.Context(), payload.Username); err != nil {
		resp.Error(w, err)
		return
	}
	// validate password
	if err = h.srv.ComparePasswords(player.Password, payload.Password); err != nil {
		resp.Error(w, err)
		return
	}
	// generate token
	if token, err = h.token.Generate(r.Context(), *player); err != nil {
		resp.Error(w, err)
		return
	}
	result := loginResponse{Token: string(token)}
	resp.JSON(w, result)

}
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var payload loginParams
	defer r.Body.Close()
	if err := req.I.Will().Bind(r, &payload).Validate(payload).Err(); err != nil {
		resp.Error(w, err)
		return
	}
	ctx := r.Context()
	player := game.Player{Username: payload.Username, Password: payload.Password}
	if err := h.srv.CreatePlayer(ctx, &player); err != nil {
		resp.Error(w, err)
		return
	}
	var (
		token auth.Token
		err   error
	)
	if token, err = h.token.Generate(ctx, player); err != nil {
		resp.Error(w, err)
		return
	}
	result := loginResponse{Token: string(token)}
	resp.JSON(w, result)
}

type roomIDResponse struct {
	ID string `json:"id"`
}

func (h *Handler) createRoom(w http.ResponseWriter, r *http.Request) {
	// 1. get the user from the context
	ctx := r.Context()
	player := Player(ctx)
	if player == nil {
		resp.Error(w, ErrUnauthenticated)
		return
	}
	// 2. create a room with the user as the creator and store this room in temporary area (Hub)
	wrd := h.wordGen.Generate(word.Length)
	log.Println(wrd)
	g := game.New(player.Username, word.New(wrd))
	room := NewRoom(g)

	// 3. return the room id
	result := roomIDResponse{ID: room.g.ID.String()}
	resp.JSON(w, result)
}

type joinRoomResponse struct {
	Token string `json:"token"`
}

// joinRoom creates a new player token used to join a room using websocket connection
func (h *Handler) joinRoom(w http.ResponseWriter, r *http.Request) {
	// get the user from the context
	ctx := r.Context()
	player := Player(ctx)
	if player == nil {
		resp.Error(w, ErrUnauthenticated)
		return
	}
	// get the room id from the url params
	id := chi.URLParam(r, "id")
	uid, err := uuid.Parse(id)
	if err != nil {
		resp.Error(w, errs.B().Code(errs.InvalidArgument).Msg("invalid parameters").Err())
		return
	}
	// find the room in the temporary area (Hub)
	Hub.mu.Lock()
	_, ok := Hub.rooms[uid]
	Hub.mu.Unlock()
	if !ok {
		resp.Error(w, errs.B().Msg("room not found").Err())
		return
	}
	// return a token for the user to join the room with ws
	token := h.srv.CreateToken(player.Username, uid)
	result := joinRoomResponse{Token: token}
	resp.JSON(w, result)
}

func (h *Handler) rooms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	player := Player(ctx)
	if player == nil {
		resp.Error(w, ErrUnauthenticated)
		return
	}
	rooms, err := h.srv.GetPlayerRooms(ctx, player.ID)
	if err != nil {
		resp.Error(w, err)
		return
	}
	resp.JSON(w, rooms) // TODO: create separate response type for this
}

func (h *Handler) room(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := uuid.Parse(id)
	if err != nil {
		resp.Error(w, errs.B().Code(errs.InvalidArgument).Msg("invalid parameters").Err())
		return
	}
	game, err := h.srv.GetGame(r.Context(), uid)
	if err != nil {
		resp.Error(w, err)
		return
	}
	resp.JSON(w, game)
	// TODO:
	// 1. get the user from the context
	// 2. get the room id from the url params
	// 3. return the game details for this room as well as the words this user played in this game
}

func (h *Handler) Stop(ctx context.Context) error {
	return h.s.Shutdown(ctx)
}
