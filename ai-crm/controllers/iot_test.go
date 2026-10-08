package controllers

import (
	"ai-crm/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFormatIoTNumberUsesEnglishCompactUnits(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{name: "plain integer", value: 999, want: "999"},
		{name: "decimal", value: 12.345, want: "12.35"},
		{name: "thousands", value: 2000, want: "2.0K"},
		{name: "millions", value: 20000000, want: "20.0M"},
		{name: "billions", value: 1500000000, want: "1.5B"},
		{name: "negative millions", value: -1861000, want: "-1.9M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatIoTNumber(tt.value); got != tt.want {
				t.Fatalf("formatIoTNumber(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestIoTFirmwareVersionFromRequestPrefersHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/iot/v1/sales-dashboard", nil)
	req.Header.Set("X-IOT-Firmware-Version", "v1.0.9.5")
	req.Header.Set("User-Agent", "v1.0.9.4")

	if got := iotFirmwareVersionFromRequest(req); got != "v1.0.9.5" {
		t.Fatalf("firmware version = %q, want v1.0.9.5", got)
	}
}

func TestIoTFirmwareVersionFromUserAgentFallback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/iot/v1/sales-dashboard", nil)
	req.Header.Set("User-Agent", "Halmify-IoT v1.0.9.4")

	if got := iotFirmwareVersionFromRequest(req); got != "v1.0.9.4" {
		t.Fatalf("firmware version = %q, want v1.0.9.4", got)
	}
}

func TestIoTFirmwareVersionFromUserAgentIgnoresBrowserVersions(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/iot/v1/sales-dashboard", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Chrome/110.0.0.0 Safari/537.36")

	if got := iotFirmwareVersionFromRequest(req); got != "" {
		t.Fatalf("firmware version = %q, want empty", got)
	}
}

func TestIoTDevicePayloadUsesTwentyFiveMinuteOnlineWindow(t *testing.T) {
	seenWithinWindow := time.Now().Add(-24 * time.Minute)
	withinPayload := iotDevicePayload(models.IoTDevice{LastSeenAt: &seenWithinWindow})
	if online, ok := withinPayload["online"].(bool); !ok || !online {
		t.Fatalf("online within 24 minutes = %v, want true", withinPayload["online"])
	}

	seenOutsideWindow := time.Now().Add(-26 * time.Minute)
	outsidePayload := iotDevicePayload(models.IoTDevice{LastSeenAt: &seenOutsideWindow})
	if online, ok := outsidePayload["online"].(bool); !ok || online {
		t.Fatalf("online after 26 minutes = %v, want false", outsidePayload["online"])
	}
}

func TestIoTDeviceActivityUpdatesSyncAndNewFirmwareVersion(t *testing.T) {
	now := time.Now()
	device := models.IoTDevice{FirmwareVersion: "v1.0.8"}
	updates := iotDeviceActivityUpdates(now, " 10.0.0.8 ", " v1.0.9.4 ", device, iotDeviceActivityOptions{
		Sync:            true,
		FirmwareVersion: "v1.0.9.4",
	})

	if got := updates["firmware_version"]; got != "v1.0.9.4" {
		t.Fatalf("firmware update = %v, want v1.0.9.4", got)
	}
	if _, ok := updates["last_sync_at"]; !ok {
		t.Fatalf("last_sync_at missing from sync updates: %#v", updates)
	}
	if got := updates["last_ip"]; got != "10.0.0.8" {
		t.Fatalf("last_ip = %v, want 10.0.0.8", got)
	}
	if got := updates["last_user_agent"]; got != "v1.0.9.4" {
		t.Fatalf("last_user_agent = %v, want v1.0.9.4", got)
	}
}

func TestIoTDeviceActivityUpdatesDoNotDowngradeFirmwareVersion(t *testing.T) {
	now := time.Now()
	device := models.IoTDevice{FirmwareVersion: "v1.0.9.5"}
	updates := iotDeviceActivityUpdates(now, "10.0.0.8", "v1.0.9.4", device, iotDeviceActivityOptions{
		FirmwareVersion: "v1.0.9.4",
	})

	if got, ok := updates["firmware_version"]; ok {
		t.Fatalf("firmware update = %v, want no downgrade", got)
	}
}

func TestIoTDeviceActivityUpdatesIgnoreEmptyFirmwareVersion(t *testing.T) {
	now := time.Now()
	device := models.IoTDevice{FirmwareVersion: "v1.0.9.4"}
	updates := iotDeviceActivityUpdates(now, "10.0.0.8", "", device, iotDeviceActivityOptions{})

	if got, ok := updates["firmware_version"]; ok {
		t.Fatalf("firmware update = %v, want no empty overwrite", got)
	}
}
