package infra

import (
	"sync"

	"anonymity/internal/config"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/redis/go-redis/v9"
)

var (
	Redis *redis.Client
	ES    *elasticsearch.Client

	once sync.Once
)

func Init(cfg *config.Config) {
	once.Do(func() {
		initRedis(cfg)
		initElastic(cfg)
	})
}
