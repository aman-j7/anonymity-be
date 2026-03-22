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
	"anonymity/internal/questions"
	"anonymity/internal/router"
	"anonymity/internal/store"
)

func Run() {

	cfg := config.Load()
	if cfg == nil {
		log.Fatal("Error on loading env")
	}

	infra.Init(cfg)

	gameStore := store.New()
	gameStore.StartCleanup(constants.CleanupInterval, constants.MaxIdleTime)

	qs := &questions.ESQuestionService{}
	qb := questions.NewQuestionBank(qs)
	openRouter := questions.InitOpenRouter(cfg.OpenRouterApiKey)
	esRepository := &es.ESRepository{}

	checkQuestionsAvailability(esRepository, openRouter, qs)
	engine := game.NewEngine(qb)

	httpHandler := handlers.NewHTTPHandler(gameStore)
	wsHandler := handlers.NewWSHandler(gameStore, engine)

	r := router.New(httpHandler, wsHandler)

	startServer(cfg.Port, r)
}

func checkQuestionsAvailability(es *es.ESRepository, openRouter *questions.OpenRouter, qs *questions.ESQuestionService) {
	cat, err := es.GetCategoriesOrFallback(20)
	if err != nil {
		log.Fatalf("Error on fetching categories %v", err)
	}
	questions, err := qs.GenerateQuestionsForAllCategories(openRouter, cat)
	if err != nil {
		log.Fatalf("Error on generating questions for categories %v", err)
	}
	err = es.BulkQuestionsPush(questions)
	if err != nil {
		log.Fatalf("Error on bulk questions for push %v", err)
	}
}

func startServer(port string, handler http.Handler) {
	log.Printf("=== Anonymity Server ===")
	log.Printf("HTTP server starting on :%s", port)
	log.Printf("Test client: http://localhost:%s", port)
	log.Printf("Health check: http://localhost:%s/api/health", port)

	log.Fatal(http.ListenAndServe(":"+port, handler))
}
