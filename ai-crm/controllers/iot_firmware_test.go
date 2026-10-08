package controllers

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestParseIoTFirmwarePackageGeneratedZip(t *testing.T) {
	path := filepath.Join("..", "esp32", "ESP32", "projects", "esp32_s3_initializer", "ota_packages", "halmify-esp32-s3-epaper-v1.0.2.zip")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("OTA package fixture is not available: %v", err)
	}

	info, err := parseIoTFirmwarePackage(filepath.Base(path), data)
	if err != nil {
		t.Fatalf("parseIoTFirmwarePackage() error = %v", err)
	}
	if info.Version != "v1.0.2" {
		t.Fatalf("version = %q, want v1.0.2", info.Version)
	}
	if info.Model != "esp32-s3-epaper" {
		t.Fatalf("model = %q, want esp32-s3-epaper", info.Model)
	}
	if info.Signature == "" {
		t.Fatal("signature is empty")
	}
	if len(info.Data) == 0 {
		t.Fatal("firmware data is empty")
	}
	if len(info.SHA256) != 64 {
		t.Fatalf("sha256 length = %d, want 64", len(info.SHA256))
	}
}

func TestParseIoTFirmwareNamePreservesDisplaySizeModel(t *testing.T) {
	version, model := parseIoTFirmwareName("halmify-esp32-s3-epaper-3in97-v1.0.9.15.zip")

	if version != "v1.0.9.15" {
		t.Fatalf("version = %q, want v1.0.9.15", version)
	}
	if model != "esp32-s3-epaper-3in97" {
		t.Fatalf("model = %q, want esp32-s3-epaper-3in97", model)
	}
}

func TestNormalizeIoTFirmwarePublishRequestDevices(t *testing.T) {
	req := iotFirmwarePublishRequest{
		SNs: []string{" sn-001 ", "SN-001", "sn-002"},
	}

	normalizeIoTFirmwarePublishRequest(&req)

	if req.Target != "devices" {
		t.Fatalf("target = %q, want devices", req.Target)
	}
	wantSNs := []string{"SN-001", "SN-002"}
	if !reflect.DeepEqual(req.SNs, wantSNs) {
		t.Fatalf("sns = %#v, want %#v", req.SNs, wantSNs)
	}
}

func TestIoTFirmwareTargetModelWhereMatchesLegacyBlankModelForDefaultRelease(t *testing.T) {
	clause, args := iotFirmwareTargetModelWhere("")
	if clause != "(model IN ? OR model = '' OR model IS NULL)" {
		t.Fatalf("clause = %q, want default model legacy fallback", clause)
	}
	if !reflect.DeepEqual(args, []interface{}{[]string{"esp32-s3-epaper", "esp32-s3", "esp32s3", "s3"}}) {
		t.Fatalf("args = %#v, want default model arg", args)
	}

	clause, args = iotFirmwareTargetModelWhere("esp32-p4-epaper")
	if clause != "model IN ?" {
		t.Fatalf("p4 clause = %q, want exact model match", clause)
	}
	if !reflect.DeepEqual(args, []interface{}{[]string{"esp32-p4-epaper", "esp32-p4", "esp32p4", "p4"}}) {
		t.Fatalf("p4 args = %#v, want p4 model arg", args)
	}
}

func TestIoTOTADownloadTokenAllowsDottedVersion(t *testing.T) {
	token := makeIoTOTADownloadToken("halmify-14c19fd469cc", 42, "v1.0.2", time.Now().Add(time.Minute))

	claims, ok := parseIoTOTADownloadToken(token)
	if !ok {
		t.Fatal("parseIoTOTADownloadToken() failed")
	}
	if claims.SN != "HALMIFY-14C19FD469CC" {
		t.Fatalf("sn = %q, want normalized SN", claims.SN)
	}
	if claims.ReleaseID != 42 {
		t.Fatalf("release = %d, want 42", claims.ReleaseID)
	}
	if claims.Version != "v1.0.2" {
		t.Fatalf("version = %q, want v1.0.2", claims.Version)
	}
	if claims.Nonce == "" {
		t.Fatal("nonce is empty")
	}
}
