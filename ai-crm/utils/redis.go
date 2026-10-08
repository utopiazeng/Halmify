package utils

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/server/web"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() error {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		host, _ := web.AppConfig.String("redis_host")
		port, _ := web.AppConfig.String("redis_port")
		if strings.TrimSpace(host) != "" && strings.TrimSpace(port) != "" {
			addr = strings.TrimSpace(host) + ":" + strings.TrimSpace(port)
		}
	}
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	password := strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	if password == "" {
		password, _ = web.AppConfig.String("redis_password")
	}

	db := 0
	if envDB := strings.TrimSpace(os.Getenv("REDIS_DB")); envDB != "" {
		if parsed, err := strconv.Atoi(envDB); err == nil {
			db = parsed
		}
	} else if confDB, err := web.AppConfig.Int("redis_db"); err == nil {
		db = confDB
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return err
	}
	RedisClient = client
	LogInfo("Redis连接成功: %s db=%d", addr, db)
	return nil
}
