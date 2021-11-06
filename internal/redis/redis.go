package redis

import (
	"log"
	"time"

	"github.com/centrifugal/centrifuge"
	"go.uber.org/zap"
)

func HandlerRedisShard(node *centrifuge.Node) {
	redisShardConfigs := []centrifuge.RedisShardConfig{
		{Address: "localhost:6379"},
		// {Address: "localhost:6380"},
	}
	var redisShards []*centrifuge.RedisShard
	for _, redisConf := range redisShardConfigs {
		redisShard, err := centrifuge.NewRedisShard(node, redisConf)
		if err != nil {
			log.Fatal("centrifuge.NewRedisShard()",
				zap.Error(err),
			)
		}
		redisShards = append(redisShards, redisShard)
	}

	broker, err := centrifuge.NewRedisBroker(node, centrifuge.RedisBrokerConfig{
		// Use reasonably large expiration interval for stream meta key,
		// much bigger than maximum HistoryLifetime value in Node config.
		// This way stream metadata will expire, in some cases you may want
		// to prevent its expiration setting this to zero value.
		HistoryMetaTTL: 7 * 24 * time.Hour,

		// And configure a couple of shards to use.
		Shards: redisShards,
	})
	if err != nil {
		log.Fatal("centrifuge.NewRedisBroker()",
			zap.Error(err),
		)
	}
	node.SetBroker(broker)

	presenceManager, err := centrifuge.NewRedisPresenceManager(node, centrifuge.RedisPresenceManagerConfig{
		Shards: redisShards,
	})
	if err != nil {
		log.Fatal("centrifuge.NewRedisPresenceManager()",
			zap.Error(err),
		)
	}
	node.SetPresenceManager(presenceManager)
}
