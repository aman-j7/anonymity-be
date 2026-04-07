package router

import (
	"net/http"

	chi "github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware" 
	"github.com/rs/cors"

	"anonymity/internal/handlers"
	mw "anonymity/internal/middleware" 
)

// New returns a chi router with HTTP + WebSocket routes, including rate limiting
func New(httpHandler *handlers.HTTPHandler, wsHandler *handlers.WSHandler, rateLimiter *mw.RateLimiter) http.Handler {
	r := chi.NewRouter()

	// ---------- Global middlewares ----------
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}).Handler)

	
	r.Post("/api/rooms", mw.HTTPRateLimiter(rateLimiter, "create_room")(httpHandler.CreateRoom))
	r.Get("/api/rooms/{code}", mw.HTTPRateLimiter(rateLimiter, "get_room")(httpHandler.GetRoom))
	r.Get("/api/health", httpHandler.Health)

	
	r.Get("/ws", wsHandler.HandleConnection)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	return r
}