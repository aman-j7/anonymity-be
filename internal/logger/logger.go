package logger

import (
	"anonymity/internal/elasticsearch"
)

func EsLogger(indexName string, data map[string]interface{}) {
	go elasticsearch.Logger(indexName, data)
}
