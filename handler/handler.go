package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lordvidex/errs"
	"github.com/lordvidex/x/auth"
	"github.com/lordvidex/x/ptr"
	"github.com/lordvidex/x/req"
	"github.com/lordvidex/x/resp"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/Chat-Map/wordle-server/handler/token"
	"github.com/Chat-Map/wordle-server/service"
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

type Handler struct {
	s      *http.Server
	router chi.Router
	srv    *service.Service
	token  token.Handler
}

func New(srv *service.Service, tokenHandler token.Handler) *Handler {
	h := &Handler{
		router: chi.NewRouter(),
		srv:    srv,
		token:  tokenHandler,
	}

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
	uid := h.srv.NewRoom(player.Username)
	result := roomIDResponse{ID: uid}
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
	_, ok := h.srv.GetRoom(uid)
	if !ok {
		resp.Error(w, errs.B().Code(errs.NotFound).Msg("room not found").Err())
		return
	}
	// return a token for the user to join the room with ws
	token := h.srv.CreateInvite(ptr.ToObj(player), uid)
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
	games := make([]game.Response, len(rooms))
	for i, g := range rooms {
		games[i] = game.ToResponse(g, player.Username)
	}
	resp.JSON(w, games)
}

func (h *Handler) room(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	player := Player(ctx)
	if player == nil {
		resp.Error(w, ErrUnauthenticated)
		return
	}
	id := chi.URLParam(r, "id")
	uid, err := uuid.Parse(id)
	if err != nil {
		resp.Error(w, errs.B().Code(errs.InvalidArgument).Msg("invalid parameters").Err())
		return
	}
	gm, err := h.srv.GetGame(ctx, player.ID, uid)
	if err != nil {
		resp.Error(w, err)
		return
	}
	resp.JSON(w, game.ToResponse(ptr.ToObj(gm), player.Username))
}

func (h *Handler) Stop(ctx context.Context) error {
	h.srv.Stop(ctx)
	return h.s.Shutdown(ctx)
}

func (h *Handler) live(w http.ResponseWriter, r *http.Request) {
	// Parse token from request query
	token := r.URL.Query().Get("token")
	p, gameID, ok := h.srv.GetInviteData(token)
	if !ok {
		resp.Error(w, errs.B().Code(errs.InvalidArgument).Msg("invalid token").Err())
		return
	}

	room, ok := h.srv.GetRoom(gameID)
	if !ok {
		resp.Error(w, errs.B().Code(errs.InvalidArgument).Msg("game not found").Err())
		return
	}

	// Check if the game has started already and user has not joined
	if err := room.CanJoin(p.Username); err != nil {
		resp.Error(w, errs.B(err).Code(errs.InvalidArgument).Err())
		return
	}

	// Upgrade the HTTP connection to a websocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading connection: %v", err)
		return
	}

	room.Join(p, conn)
}
