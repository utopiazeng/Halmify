package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

type SchedulerLocker struct {
	rdb      *redis.Client
	ttl      time.Duration
	renewSec time.Duration
}

func NewSchedulerLocker(rdb *redis.Client, ttl, renew time.Duration) *SchedulerLocker {
	if ttl <= 0 {
		ttl = 90 * time.Second
	}
	if renew <= 0 {
		renew = 30 * time.Second
	}
	return &SchedulerLocker{rdb: rdb, ttl: ttl, renewSec: renew}
}

func (l *SchedulerLocker) WithLock(ctx context.Context, key string, fn func()) {
	if l == nil || l.rdb == nil {
		return
	}
	token := lockToken()
	ok, err := l.rdb.SetNX(ctx, key, token, l.ttl).Result()
	if err != nil || !ok {
		return
	}
	defer l.release(ctx, key, token)

	stop := make(chan struct{})
	defer close(stop)
	go l.renewLoop(ctx, key, token, stop)
	fn()
}

func (l *SchedulerLocker) renewLoop(ctx context.Context, key, token string, stop <-chan struct{}) {
	ticker := time.NewTicker(l.renewSec)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = l.extend(ctx, key, token)
		case <-stop:
			return
		}
	}
}

func (l *SchedulerLocker) extend(ctx context.Context, key, token string) error {
	const script = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("pexpire", KEYS[1], ARGV[2]) else return 0 end`
	return l.rdb.Eval(ctx, script, []string{key}, token, int64(l.ttl/time.Millisecond)).Err()
}

func (l *SchedulerLocker) release(ctx context.Context, key, token string) {
	const script = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	_ = l.rdb.Eval(ctx, script, []string{key}, token).Err()
}

func lockToken() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return time.Now().Format(time.RFC3339Nano)
	}
	return hex.EncodeToString(buf[:])
}
