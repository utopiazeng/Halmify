package services

import (
	"ai-crm/utils"
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"
)

const imPubSubChannel = "im:events:global"

type IMEvent struct {
	Event     string      `json:"event"`
	TenantID  uint        `json:"tenant_id"`
	UserID    uint        `json:"user_id"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type imEventPayload struct {
	Event     string          `json:"event"`
	TenantID  uint            `json:"tenant_id"`
	UserID    uint            `json:"user_id"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

type imRealtimeHub struct {
	mu        sync.RWMutex
	listeners map[string][]chan IMEvent
}

var realtimeHub = &imRealtimeHub{
	listeners: map[string][]chan IMEvent{},
}

func listenerKeyString(tenantID, userID uint) string {
	return fmtUint(tenantID) + ":" + fmtUint(userID)
}

func fmtUint(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}

func InitIMRealtime() {
	if utils.RedisClient == nil {
		return
	}
	go func() {
		ctx := context.Background()
		sub := utils.RedisClient.Subscribe(ctx, imPubSubChannel)
		defer sub.Close()
		ch := sub.Channel()
		for msg := range ch {
			var payload imEventPayload
			if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
				continue
			}
			realtimeHub.dispatch(IMEvent{
				Event:     payload.Event,
				TenantID:  payload.TenantID,
				UserID:    payload.UserID,
				Timestamp: payload.Timestamp,
				Data:      payload.Data,
			})
		}
	}()
}

func (h *imRealtimeHub) dispatch(event IMEvent) {
	key := listenerKeyString(event.TenantID, event.UserID)
	h.mu.RLock()
	targets := append([]chan IMEvent{}, h.listeners[key]...)
	h.mu.RUnlock()
	for _, ch := range targets {
		select {
		case ch <- event:
		default:
		}
	}
}

func RegisterIMListener(tenantID, userID uint) (chan IMEvent, func()) {
	ch := make(chan IMEvent, 64)
	key := listenerKeyString(tenantID, userID)
	realtimeHub.mu.Lock()
	realtimeHub.listeners[key] = append(realtimeHub.listeners[key], ch)
	realtimeHub.mu.Unlock()
	cleanup := func() {
		realtimeHub.mu.Lock()
		defer realtimeHub.mu.Unlock()
		list := realtimeHub.listeners[key]
		next := make([]chan IMEvent, 0, len(list))
		for _, item := range list {
			if item != ch {
				next = append(next, item)
			}
		}
		if len(next) == 0 {
			delete(realtimeHub.listeners, key)
		} else {
			realtimeHub.listeners[key] = next
		}
		close(ch)
	}
	return ch, cleanup
}

func PublishIMEvent(event string, tenantID, userID uint, data interface{}) {
	payload := IMEvent{
		Event:     event,
		TenantID:  tenantID,
		UserID:    userID,
		Timestamp: time.Now().UnixMilli(),
		Data:      data,
	}
	realtimeHub.dispatch(payload)
	if utils.RedisClient == nil {
		return
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = utils.RedisClient.Publish(context.Background(), imPubSubChannel, b).Err()
}
