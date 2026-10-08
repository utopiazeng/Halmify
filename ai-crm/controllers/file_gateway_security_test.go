package controllers

import (
	"bufio"
	"context"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestCopyUploadWithLimitRejectsOversize(t *testing.T) {
	err := copyUploadWithLimit(io.Discard, strings.NewReader("123456"), 5)
	if err != errUploadLimitExceeded {
		t.Fatalf("expected upload limit error, got %v", err)
	}
}

func TestCopyUploadWithLimitAllowsWithinLimit(t *testing.T) {
	err := copyUploadWithLimit(io.Discard, strings.NewReader("12345"), 5)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestApplyDownloadLimitWithRedisBlocksRateAndConcurrency(t *testing.T) {
	client, cleanup := startTestRedis(t)
	defer cleanup()
	ctx := context.Background()
	cfg := downloadLimitConfig{
		Window:            time.Minute,
		UserRate:          1,
		IPRate:            100,
		TenantRate:        100,
		UserConcurrency:   1,
		IPConcurrency:     100,
		TenantConcurrency: 100,
		ConcurrencyTTL:    time.Minute,
		FailOpen:          true,
	}
	subject := downloadLimitSubject{TenantID: 10, UserID: 20, IP: "127.0.0.1"}

	first, err := applyDownloadLimitWithRedis(ctx, client, cfg, subject)
	if err != nil {
		t.Fatalf("unexpected first acquire error: %v", err)
	}
	if !first.Allowed || first.ReleaseFunc == nil {
		t.Fatalf("expected first request allowed with release, got %+v", first)
	}
	defer first.ReleaseFunc()

	second, err := applyDownloadLimitWithRedis(ctx, client, cfg, subject)
	if err != nil {
		t.Fatalf("unexpected second acquire error: %v", err)
	}
	if second.Allowed || second.RetryAfter <= 0 {
		t.Fatalf("expected second request blocked by rate, got %+v", second)
	}

	cfg.UserRate = 100
	third, err := applyDownloadLimitWithRedis(ctx, client, cfg, subject)
	if err != nil {
		t.Fatalf("unexpected third acquire error: %v", err)
	}
	if third.Allowed {
		if third.ReleaseFunc != nil {
			third.ReleaseFunc()
		}
		t.Fatal("expected third request blocked by concurrency")
	}

	first.ReleaseFunc()
	fourth, err := applyDownloadLimitWithRedis(ctx, client, cfg, subject)
	if err != nil {
		t.Fatalf("unexpected fourth acquire error: %v", err)
	}
	if !fourth.Allowed {
		t.Fatalf("expected fourth request allowed after release, got %+v", fourth)
	}
	if fourth.ReleaseFunc != nil {
		fourth.ReleaseFunc()
	}
}

func startTestRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()
	port := freeTCPPort(t)
	cmd := exec.Command("redis-server", "--save", "", "--appendonly", "no", "--port", strconv.Itoa(port), "--bind", "127.0.0.1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("redis stdout pipe: %v", err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("start redis-server: %v", err)
	}
	ready := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "Ready to accept connections") {
				close(ready)
				return
			}
		}
	}()
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("redis-server did not become ready")
	}
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:" + strconv.Itoa(port)})
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("ping redis-server: %v", err)
	}
	cleanup := func() {
		_ = client.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	return client, cleanup
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate tcp port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}
