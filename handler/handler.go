package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lordvidex/x/req"
	"github.com/lordvidex/x/resp"
)

type Handler struct {
	router chi.Router
}

func New() *Handler {
	h := &Handler{
		router: chi.NewRouter(),
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
		// add authentication middleware
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

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var payload loginParams
	defer r.Body.Close()
	if err := req.I.Will().Bind(r, &payload).Validate(payload).Err(); err != nil {
		resp.Error(w, err)
		return
	}
	// TODO: call login service
}
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var payload loginParams
	defer r.Body.Close()
	if err := req.I.Will().Bind(r, &payload).Validate(payload).Err(); err != nil {
		resp.Error(w, err)
		return
	}
	// TODO: call register service
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
	// TODO:
	// 1. get the user from the context
	// 2. return all the rooms for this user sorted by finished time (NULLS first)
}
func (h *Handler) room(w http.ResponseWriter, r *http.Request) {
	// TODO:
	// 1. get the user from the context
	// 2. get the room id from the url params
	// 3. return the game details for this room as well as the words this user played in this game
}
func (h *Handler) live(w http.ResponseWriter, r *http.Request) {
	// TODO:
	// 1. get the parameters from the url
	// 2. check the token to get the user's details
	// 3. upgrade the connection to websocket
	// 4. create a new connection for this user and add him to the room in the hub
}
