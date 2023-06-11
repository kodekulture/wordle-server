package handler

import (
	"net/http"

	"github.com/gorilla/mux"
)

type Handler struct {
	r *mux.Router
}

func New() *Handler {
	h := &Handler{
		r: mux.NewRouter(),
	}

	h.setup()
	return h
}

func (h *Handler) Start(port string) error {
	return http.ListenAndServe(port, h.r)
}

func (h *Handler) setup() {
	r := h.r

	r.HandleFunc("/login", h.login).Methods("POST")
	r.HandleFunc("/register", h.register).Methods("POST")
	r.HandleFunc("/create/room", h.createRoom).Methods("POST")

	r.HandleFunc("/join/room/{id}", h.joinRoom).Methods("GET")
	r.HandleFunc("/room", h.rooms).Methods("GET")
	r.HandleFunc("/room/{id}", h.room).Methods("GET")

	r.HandleFunc("/live", h.live).Methods("GET")
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}
func (h *Handler) createRoom(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}
func (h *Handler) joinRoom(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}
func (h *Handler) rooms(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}
func (h *Handler) room(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}
func (h *Handler) live(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}
