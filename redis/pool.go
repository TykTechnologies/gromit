package redis

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/rs/zerolog/log"
)

var (
	Pool *redis.Pool
)

func InitPool(host string) {
	Pool = newPool(host)
	cleanupHook()
}

func newPool(server string) *redis.Pool {

	return &redis.Pool{

		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,

		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				log.Fatal().Err(err).Msgf("could not connect to %s", server)
			}
			password := os.Getenv("REDISCLI_AUTH")
			if password != "" {
				_, err = c.Do("AUTH", password)
				if err != nil {
					log.Fatal().Err(err).Msgf("could not auth to %s", server)
				}
			}
			return c, err
		},

		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func cleanupHook() {

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGKILL)
	go func() {
		<-c
		Pool.Close()
		os.Exit(0)
	}()
}
