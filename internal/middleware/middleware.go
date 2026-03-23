package middleware

import (
	"anonymity/internal/es"
	"anonymity/internal/questions"
	"log"
)

type MiddlewareService struct {
}

func (service *MiddlewareService) CheckQuestionsAvailability(es *es.ESRepository, openRouter *questions.OpenRouter, qs *questions.ESQuestionService) {
	cat, fetchAllCategories, err := es.GetCategoriesOrFallback(20)
	if err != nil {
		log.Fatalf("Error on fetching categories %v", err)
	}
	if cat == nil && fetchAllCategories == false {
		return
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
