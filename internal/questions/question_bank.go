package questions

import (
	"anonymity/internal/models"
	"log"
)

type QuestionBank struct {
	service QuestionService
}

func NewQuestionBank(service QuestionService) *QuestionBank {
	return &QuestionBank{
		service: service,
	}
}
func (qb *QuestionBank) InitQuestionPool(room *models.Room) error {
	room.UsedQuestionIDs = make(map[string]bool)
	room.Questions = []models.Question{}
	room.QuestionIdx = 0

	//prefetching more data two times more
	initialSize := room.Settings.NumRounds * 2
	log.Println("inside InitQuestionPool -> "+(string)(len(room.Questions)))
	return qb.RefillPool(room, initialSize)
}

func (qb *QuestionBank) FetchUniqueQuestions(n int,used map[string]bool,) ([]models.Question, error) {

	result := make([]models.Question, 0, n)
	for len(result) < n {
		qs, err := qb.service.GetRandomQuestions(n * 2)
		if err != nil {
			return nil, err
		}

		for _, q := range qs {
			if !used[q.ID] {
				used[q.ID] = true
				result = append(result, q)

				if len(result) == n {
					break
				}
			}
		}

		if len(qs) == 0 {
			break
		}
	}

	return result, nil
}

func (qb *QuestionBank) RefillPool(room *models.Room, count int) error {
	newQs, err := qb.FetchUniqueQuestions(count, room.UsedQuestionIDs)
	if err != nil {
		return err
	}

	room.Questions = append(room.Questions, newQs...)
	log.Println("inside RefillPool -> "+(string)(len(room.Questions)))
	return nil
}

func (qb *QuestionBank) GetNextQuestion(room *models.Room) (*models.Question, error) {

	if room.QuestionIdx >= len(room.Questions) {
		err := qb.RefillPool(room, room.Settings.NumRounds*2)
		if err != nil {
			return nil, err
		}
	}

	q := &room.Questions[room.QuestionIdx]
	room.QuestionIdx++

	return q, nil
}
