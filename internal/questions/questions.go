package questions

import "anonymity/internal/models"

type QuestionService interface {
	GetRandomQuestions(n int) ([]models.Question, error)
}
