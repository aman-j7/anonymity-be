package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"time"
	"anonymity/internal/infra"
)

func EsLogger(indexName string, data map[string]interface{}) {
	if infra.ES == nil {
		log.Println("Elasticsearch client not initialized")
		return
	}

	
	if _, ok := data["@timestamp"]; !ok {
		data["@timestamp"] = time.Now().UTC()
	}

	body, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling log data: %v", err)
		return
	}

	res, err := infra.ES.Index(
		indexName,
		bytes.NewReader(body),
		infra.ES.Index.WithContext(context.Background()),
	)

	if err != nil {
		log.Printf("Error indexing document: %v", err)
		return
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Printf("Elasticsearch indexing error: %s", res.String())
	}
}