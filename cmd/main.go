package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"

	"anonymity/internal/config"
	"anonymity/internal/game"
	"anonymity/internal/handlers"
	"anonymity/internal/store"
)

func main() {
	cfg := config.Load()

	config.InitRedis()
	if config.RedisClient == nil {
		return
	}

	gameStore := store.New()
	gameStore.StartCleanup(cfg.CleanupInterval, cfg.MaxIdleTime)

	engine := game.NewEngine()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}).Handler)

	httpHandler := handlers.NewHTTPHandler(gameStore)
	r.Post("/api/rooms", httpHandler.CreateRoom)
	r.Get("/api/rooms/{code}", httpHandler.GetRoom)
	r.Get("/api/health", httpHandler.Health)

	wsHandler := handlers.NewWSHandler(gameStore, engine)
	r.Get("/ws", wsHandler.HandleConnection)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Printf("=== Anonymity Server ===")
	log.Printf("HTTP server starting on :%s", cfg.Port)
	log.Printf("Test client: http://localhost:%s", cfg.Port)
	log.Printf("Health check: http://localhost:%s/api/health", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}
