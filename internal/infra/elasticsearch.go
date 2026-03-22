package infra

import (
	"anonymity/internal/config"
	"context"
	"anonymity/constants"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)


func initElastic(cfg *config.Config) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{cfg.ElasticURL},
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: 5 * time.Second,
		},
	})

	if err != nil {
		log.Fatalf("Elasticsearch init failed: %v", err)
	}

	res, err := es.Info()
	if err != nil {
		log.Fatalf("Elasticsearch not reachable: %v", err)
	}
	defer res.Body.Close()

	ES = es
	log.Println("✅ Elasticsearch initialized")

	createIndexIfNotExists(constants.EsRoomLoggerIdx)
}


func createIndexIfNotExists(indexName string) {
	if indexName == "" {
		log.Println("indexName is empty")
		return
	}

	if ES == nil {
		log.Println("ES client is nil")
		return
	}

	res, err := ES.Indices.Exists([]string{indexName})
	if err != nil {
		log.Printf("Error checking index existence: %v", err)
		return
	}
	defer res.Body.Close()

	
	if res.StatusCode == 200 {
		log.Println("Index already exists:", indexName)
		return
	}

	
	if res.StatusCode != 404 {
		log.Printf("Unexpected status checking index: %d", res.StatusCode)
		return
	}

	mapping := `{
		"settings": {
			"number_of_shards": 1,
			"number_of_replicas": 0
		},
		"mappings": {
			"properties": {
				"@timestamp": { "type": "date" },
				"created_at": { "type": "date" },
				"event": { "type": "keyword" },
				"room_code": { "type": "keyword" },
				"host_id": { "type": "keyword" },
				"host_name": { "type": "text" },
				"player_count": { "type": "integer" },
				"status": { "type": "keyword" },
				"user_id": { "type": "keyword" },
				"user_name": { "type": "text" },
				"service": { "type": "keyword" }
			}
		}
	}`

	createRes, err := ES.Indices.Create(
		indexName,
		ES.Indices.Create.WithContext(context.Background()),
		ES.Indices.Create.WithBody(strings.NewReader(mapping)),
	)

	if err != nil {
		log.Printf("Error creating index: %v", err)
		return
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		log.Printf("Index creation failed: %s", createRes.String())
		return
	}

	log.Println("Index created:", indexName)
}
