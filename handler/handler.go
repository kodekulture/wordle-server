package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lordvidex/x/auth"
	"github.com/lordvidex/x/req"
	"github.com/lordvidex/x/resp"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/Chat-Map/wordle-server/handler/token"
	"github.com/Chat-Map/wordle-server/service"
)

type Handler struct {
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
	return http.ListenAndServe(port, h.router)
}

func (h *Handler) setup() {
	r := h.router
	// Public routes
	r.Group(func(r chi.Router) {
		r.Post("/login", h.login)
		r.Post("/register", h.register)
		r.Get("/live", h.live)
	})

	// Private routes
	r.Group(func(r chi.Router) {
		r.Use(h.authMiddleware(AuthDecodeTypeFetch))

		r.Post("/create/room", h.createRoom)
		r.Get("/join/room/{id}", h.joinRoom)
		r.Get("/room", h.rooms)
		r.Get("/room/{id}", h.room)
	})

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

func (h *Handler) createRoom(w http.ResponseWriter, r *http.Request) {
	// TODO:
	// 1. get the user from the context
	//
	// 2. create a room with the user as the creator and store this room in temporary area (Hub)
	//
	// 3. return the room id
}
func (h *Handler) joinRoom(w http.ResponseWriter, r *http.Request) {
	// TODO:
	// 1. get the user from the context
	// 2. get the room id from the url params
	// 3. find the room in the temporary area (Hub)
	// 4a. if room does not exist return error
	// 4b. if room exists, return a token for the user to join the room with ws
}
func (h *Handler) rooms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	player := PlayerFromCtx(ctx)
	if player == nil {
		resp.Error(w, ErrUnauthenticated)
		return
	}
	rooms, err := h.srv.GetPlayerRooms(ctx, player.ID)
	if err != nil {
		resp.Error(w, err)
	}
	resp.JSON(w, rooms) // TODO: create separate response type for this
}
func (h *Handler) room(w http.ResponseWriter, r *http.Request) {
	// TODO:
	// 1. get the user from the context
	// 2. get the room id from the url params
	// 3. return the game details for this room as well as the words this user played in this game
}
