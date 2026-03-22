package es

import (
	"anonymity/constants"
	"anonymity/internal/infra"
	"anonymity/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (repo *ESRepository) getCategoriesWithLowCount(limit int) ([]string, error) {

	query := `{
	  "size": 0,
	  "aggs": {
		"categories": {
		  "terms": {
			"field": "category",
			"size": 100
		  },
		  "aggs": {
			"filter_count": {
			  "bucket_selector": {
				"buckets_path": {
				  "count": "_count"
				},
				"script": "params.count < 20"
			  }
			}
		  }
		}
	  }
	}`

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

	var raw map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}

	aggs, ok := raw["aggregations"].(map[string]interface{})
	if !ok {
		return []string{}, nil
	}

	categoriesAgg, ok := aggs["categories"].(map[string]interface{})
	if !ok {
		return []string{}, nil
	}

	buckets, ok := categoriesAgg["buckets"].([]interface{})
	if !ok || len(buckets) == 0 {
		return []string{}, nil
	}

	var categories []string
	for _, b := range buckets {
		bucket, ok := b.(map[string]interface{})
		if !ok {
			continue
		}

		key, ok := bucket["key"].(string)
		if !ok {
			continue
		}

		categories = append(categories, key)

		if limit > 0 && len(categories) >= limit {
			break
		}
	}

	return categories, nil
}

func (repo *ESRepository) isQuestionIndexEmpty() (bool, error) {

	res, err := infra.ES.Count(
		infra.ES.Count.WithContext(context.Background()),
		infra.ES.Count.WithIndex(constants.EsQuestionsIdx),
	)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return false, fmt.Errorf("ES error: %s", res.String())
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return false, err
	}

	count, ok := raw["count"].(float64)
	if !ok {
		return true, nil
	}

	return count == 0, nil
}

func (repo *ESRepository) GetCategoriesOrFallback(limit int) ([]string, error) {

	cats, err := repo.getCategoriesWithLowCount(limit)
	if err != nil {
		return nil, err
	}

	if len(cats) == 0 {
		isEmpty, err := repo.isQuestionIndexEmpty()
		if err != nil {
			return nil, err
		}

		if isEmpty {
			fmt.Println("Index is empty 🚨 Seeding questions")
		} else {
			fmt.Println("All categories already have enough questions 🔥")
		}
		return nil, nil
	}

	return cats, nil
}

func (repo *ESRepository) buildBulkRequest(questions []models.Question) (*bytes.Buffer, error) {

	var buf bytes.Buffer

	for _, q := range questions {

		meta := map[string]map[string]string{
			"index": {
				"_index": constants.EsQuestionsIdx,
				"_id":    q.ID,
			},
		}

		doc := map[string]string{
			"template": q.Template,
			"category": q.Category,
		}

		metaJSON, err := json.Marshal(meta)
		if err != nil {
			return nil, err
		}

		docJSON, err := json.Marshal(doc)
		if err != nil {
			return nil, err
		}

		buf.Write(metaJSON)
		buf.WriteByte('\n')
		buf.Write(docJSON)
		buf.WriteByte('\n')
	}

	return &buf, nil
}

func (repo *ESRepository) bulkInsertQuestions(ctx context.Context, body *bytes.Buffer) error {

	res, err := infra.ES.Bulk(
		body,
		infra.ES.Bulk.WithContext(ctx),
		infra.ES.Bulk.WithIndex(constants.EsQuestionsIdx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk insert failed: %s", res.String())
	}

	return nil
}

func (repo *ESRepository) BulkQuestionsPush(questions []models.Question) error {
	ctx := context.Background()

	bulkBody, _ := repo.buildBulkRequest(questions)

	err := repo.bulkInsertQuestions(ctx, bulkBody)

	return err
}
