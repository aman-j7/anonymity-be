package app

import (
	"log"
	"net/http"

	"anonymity/internal/config"
	"anonymity/internal/game"
	"anonymity/internal/handlers"
	"anonymity/internal/infra"
	"anonymity/internal/router"
	"anonymity/internal/store"
)

func Run() {
	// ✅ Load config
	cfg := config.Load()
	if cfg == nil {
		log.Fatal("Error on loading env")
	}

	// ✅ Init global infra
	infra.Init(cfg)

	// ✅ Core components
	gameStore := store.New()
	gameStore.StartCleanup(cfg.CleanupInterval, cfg.MaxIdleTime)

	engine := game.NewEngine()

	// ✅ Handlers
	httpHandler := handlers.NewHTTPHandler(gameStore)
	wsHandler := handlers.NewWSHandler(gameStore, engine)

	// ✅ Router
	r := router.New(httpHandler, wsHandler)

	// ✅ Start server
	startServer(cfg.Port, r)
}

func startServer(port string, handler http.Handler) {
	log.Printf("=== Anonymity Server ===")
	log.Printf("HTTP server starting on :%s", port)
	log.Printf("Test client: http://localhost:%s", port)
	log.Printf("Health check: http://localhost:%s/api/health", port)

	log.Fatal(http.ListenAndServe(":"+port, handler))
}
