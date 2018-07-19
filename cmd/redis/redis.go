package redis

import (
	"fmt"
	"time"

	"github.com/go-redis/cache"
	"github.com/go-redis/redis"
	"github.com/vmihailenco/msgpack"
)

// New redis cache reference
func New() *cache.Codec {
	// set up redis cache
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"server1": ":6379",
		},
	})
	codec := &cache.Codec{
		Redis: ring,

		Marshal: func(v interface{}) ([]byte, error) {
			return msgpack.Marshal(v)
		},
		Unmarshal: func(b []byte, v interface{}) error {
			return msgpack.Unmarshal(b, v)
		},
	}
	return codec
}

// Key formats the values into a consistent key format
func Key(from, to string, date time.Time) string {
	return fmt.Sprintf("%v%v%v", from, to, date.Unix())
}
