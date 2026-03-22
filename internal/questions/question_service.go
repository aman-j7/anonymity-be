package questions

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"

	"anonymity/constants"
	"anonymity/internal/infra"
	"anonymity/internal/models"
)

type ESQuestionService struct{}

type esHit struct {
	ID     string          `json:"_id"`
	Source models.Question `json:"_source"`
}

type esResponse struct {
	Hits struct {
		Hits []esHit `json:"hits"`
	} `json:"hits"`
}

func (s *ESQuestionService) GetRandomQuestions(n int) ([]models.Question, error) {

	query := fmt.Sprintf(`{
		"size": %d,
		"query": {
		  "match_all": {}
		}
	  }`, n)
	  

	res, err := infra.ES.Search(
		infra.ES.Search.WithContext(context.Background()),
		infra.ES.Search.WithIndex(constants.EsQuestionsIdx),
		infra.ES.Search.WithBody(strings.NewReader(query)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("ES error: %s", res.String())
	}

	var r esResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}

	questions := make([]models.Question, 0, len(r.Hits.Hits))

	rand.Shuffle(len(questions), func(i, j int) {
		questions[i], questions[j] = questions[j], questions[i]
	})

	for _, hit := range r.Hits.Hits {
		q := hit.Source
		q.ID = hit.ID
		questions = append(questions, q)
	}

	return questions, nil
}
