package services

import (
	"ai-crm/utils"
	"context"
	"os"
	"strconv"
	"time"

	"github.com/beego/beego/v2/server/web"
)

var (
	lockValue = "1"
)

func lockTTL() time.Duration {
	sec := web.AppConfig.DefaultInt("scheduler_lock_ttl_sec", 90)
	if env := os.Getenv("SCHEDULER_LOCK_TTL_SEC"); env != "" {
		if parsed, err := strconv.Atoi(env); err == nil && parsed > 0 {
			sec = parsed
		}
	}
	if sec <= 0 {
		sec = 90
	}
	return time.Duration(sec) * time.Second
}

func lockRenewInterval() time.Duration {
	sec := web.AppConfig.DefaultInt("scheduler_lock_renew_sec", 30)
	if env := os.Getenv("SCHEDULER_LOCK_RENEW_SEC"); env != "" {
		if parsed, err := strconv.Atoi(env); err == nil && parsed > 0 {
			sec = parsed
		}
	}
	if sec <= 0 {
		sec = 30
	}
	return time.Duration(sec) * time.Second
}

func withJobLock(jobName string, fn func()) {
	if utils.RedisClient == nil {
		return
	}
	ctx := context.Background()
	key := "sched:lock:" + jobName
	ttl := lockTTL()
	ok, err := utils.RedisClient.SetNX(ctx, key, lockValue, ttl).Result()
	if err != nil || !ok {
		return
	}
	defer utils.RedisClient.Del(ctx, key)

	stop := make(chan struct{})
	defer close(stop)

	go renewLoop(ctx, key, stop)
	fn()
}

func WithSchedulerLock(jobName string, fn func()) {
	withJobLock(jobName, fn)
}

func renewLoop(ctx context.Context, key string, stop <-chan struct{}) {
	ticker := time.NewTicker(lockRenewInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = utils.RedisClient.Expire(ctx, key, lockTTL()).Err()
		case <-stop:
			return
		}
	}
}
