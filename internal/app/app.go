package app

import (
	"log"
	"net/http"

	"anonymity/constants"
	"anonymity/internal/config"
	"anonymity/internal/es"
	"anonymity/internal/game"
	"anonymity/internal/handlers"
	"anonymity/internal/infra"
	"anonymity/internal/middleware"
	"anonymity/internal/questions"
	"anonymity/internal/router"
	"anonymity/internal/store"
)

func Run() {

	cfg := config.Load()
	infra.Init(cfg)

	qs := &questions.ESQuestionService{}
	qb := questions.NewQuestionBank(qs)
	esRepository := &es.ESRepository{}
	middleware := &middleware.MiddlewareService{}

	openRouter := questions.InitOpenRouter(cfg.OpenRouterApiKey)
	middleware.CheckQuestionsAvailability(esRepository, openRouter, qs)
	engine := game.NewEngine(qb)

	gameStore := store.New()
	gameStore.StartCleanup(constants.CleanupInterval, constants.MaxIdleTime)

	httpHandler := handlers.NewHTTPHandler(gameStore)
	wsHandler := handlers.NewWSHandler(gameStore, engine)

	router := router.New(httpHandler, wsHandler)

	startServer(cfg.Port, router)
}

func startServer(port string, handler http.Handler) {
	log.Printf("=== Anonymity Server ===")
	log.Printf("HTTP server starting on :%s", port)
	log.Printf("Test client: http://localhost:%s", port)
	log.Printf("Health check: http://localhost:%s/api/health", port)

	log.Fatal(http.ListenAndServe(":"+port, handler))
}
