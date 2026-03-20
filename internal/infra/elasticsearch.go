package infra

import (
	"anonymity/internal/config"
	"log"
	"net/http"
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
}
