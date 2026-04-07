package app

import (
	"log"
	"net/http"
	"time"

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
	"github.com/redis/go-redis/v9"
)

func Run() {

	cfg := config.Load()
	infra.Init(cfg)

	qs := &questions.ESQuestionService{}
	qb := questions.NewQuestionBank(qs)
	esRepository := &es.ESRepository{}

	mwService := &middleware.MiddlewareService{}

	openRouter := questions.InitOpenRouter(cfg.OpenRouterApiKey)
	mwService.CheckQuestionsAvailability(esRepository, openRouter, qs)

	engine := game.NewEngine(qb)

	gameStore := store.New()
	gameStore.StartCleanup(constants.CleanupInterval, constants.MaxIdleTime)


	httpHandler := handlers.NewHTTPHandler(gameStore)
	rateLimiter := newDefaultRateLimiter(infra.Redis)
	wsHandler := handlers.NewWSHandler(gameStore, engine, rateLimiter)
	router := router.New(httpHandler, wsHandler, rateLimiter)

	startServer(cfg.Port, router)
}

func startServer(port string, handler http.Handler) {
	log.Printf("=== Anonymity Server ===")
	log.Printf("HTTP server starting on :%s", port)
	log.Printf("Test client: http://localhost:%s", port)
	log.Printf("Health check: http://localhost:%s/api/health", port)

	log.Fatal(http.ListenAndServe(":"+port, handler))
}
func newDefaultRateLimiter(redisClient *redis.Client) *middleware.RateLimiter {
	return middleware.NewRateLimiter(redisClient, map[string]middleware.ActionLimit{
		
		"submit_answer": {Limit: 2, Window: 10 * time.Second},
		"submit_vote":   {Limit: 3, Window: 10 * time.Second},
		"emoji_react":   {Limit: 10, Window: 5 * time.Second},
		"start_game":    {Limit: 1, Window: 10 * time.Second},

		"create_room": {Limit: 3, Window: 10 * time.Second},
		"get_room":    {Limit: 5, Window: 5 * time.Second},
	})
}
