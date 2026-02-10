package cryptodaemon

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdbpass = os.Getenv("REDIS_PASS")
var ctx = context.Background()
var cryptoDb = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	DB:       14,
	Password: rdbpass,
})

func main() {
	fmt.Println("Crypto exchange watcher started!")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		cursor := uint64(0)
		keys, _, err := cryptoDb.Scan(ctx, cursor, "ffio:*", 500).Result()
		if err != nil {
			fmt.Printf("Error scanning keys: %v\n", err)
			return
		}

		for _, key := range keys {
			ttl := cryptoDb.TTL(ctx, key).Val()
			if ttl > 0 {
				token, err := cryptoDb.Get(ctx, key).Result()
				if err != nil {
					fmt.Printf("Error getting key %s: %v\n", key, err)
					continue
				}
				fmt.Printf("%s", token)
			}
		}
	}
}
