package redis

import (
	"goframe/pkg/confer"
	"time"

	"github.com/gomodule/redigo/redis"
)

var redisPool *redis.Pool

func InitRedis(conf confer.Redis) *redis.Pool {
	redisPool = &redis.Pool{
		MaxIdle:     1,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", conf.Address,
				redis.DialConnectTimeout(time.Second*5),
				redis.DialReadTimeout(time.Second*5),
				redis.DialWriteTimeout(time.Second*5))
			if err != nil {
				return nil, err
			}
			return c, err
		},
		//TestOnBorrow: func(c redis.Conn, t time.Time) error {
		//	_, err := c.Do("PING")
		//	return err
		//},
	}
	return redisPool
}

func getRedisPool() *redis.Pool {
	return redisPool
}
