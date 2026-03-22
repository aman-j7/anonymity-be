package logger

import "anonymity/internal/es"

func EsLogger(indexName string, data map[string]interface{}) {
	go es.Logger(indexName, data)
}
