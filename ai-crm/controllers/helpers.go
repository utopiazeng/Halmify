package controllers

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/server/web/context"
)

func queryInt(ctx *context.Context, key string, fallback int) int {
	value := strings.TrimSpace(ctx.Input.Query(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func queryUint(ctx *context.Context, key string) uint {
	value := strings.TrimSpace(ctx.Input.Query(key))
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return uint(parsed)
}

func queryFloat(ctx *context.Context, key string, fallback float64) float64 {
	value := strings.TrimSpace(ctx.Input.Query(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func pathUint(ctx *context.Context, key string) (uint, bool) {
	value := strings.TrimSpace(ctx.Input.Param(key))
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, false
	}
	return uint(parsed), true
}

func toJSONString(value interface{}) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func normalizeTags(value interface{}) string {
	switch tags := value.(type) {
	case nil:
		return ""
	case string:
		return tags
	case []interface{}:
		parts := make([]string, 0, len(tags))
		for _, item := range tags {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				parts = append(parts, strings.TrimSpace(text))
			}
		}
		return strings.Join(parts, ",")
	case []string:
		return strings.Join(tags, ",")
	default:
		return toJSONString(tags)
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
