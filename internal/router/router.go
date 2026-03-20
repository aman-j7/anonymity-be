package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"

	"anonymity/internal/handlers"
)

func New(httpHandler *handlers.HTTPHandler, wsHandler *handlers.WSHandler) http.Handler {
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}).Handler)

	// HTTP routes
	r.Post("/api/rooms", httpHandler.CreateRoom)
	r.Get("/api/rooms/{code}", httpHandler.GetRoom)
	r.Get("/api/health", httpHandler.Health)

	// WebSocket
	r.Get("/ws", wsHandler.HandleConnection)

	// Static
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	return r
}
