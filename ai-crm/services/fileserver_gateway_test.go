package services

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEvaluateNodeReasons(t *testing.T) {
	now := time.Now()
	cacheTTL := 15 * time.Second
	minFree := uint64(100)
	hwm := 85.0

	node := FileserverNode{
		ID:             "fs-1",
		Weight:         testNodeWeight(),
		Healthy:        true,
		LastCheckedAt:  now,
		FreeBytes:      200,
		TotalBytes:     1000,
		UsedBytes:      800,
		StoragePercent: 80,
	}
	if reasons := evaluateNodeReasons(node, now, cacheTTL, true, minFree, hwm); len(reasons) != 0 {
		t.Fatalf("expected no reasons, got %v", reasons)
	}

	node.Healthy = false
	reasons := evaluateNodeReasons(node, now, cacheTTL, true, minFree, hwm)
	if !containsReason(reasons, NodeReasonUnhealthy) {
		t.Fatalf("expected %s, got %v", NodeReasonUnhealthy, reasons)
	}

	node.Healthy = true
	node.LastCheckedAt = now.Add(-time.Minute)
	reasons = evaluateNodeReasons(node, now, cacheTTL, true, minFree, hwm)
	if !containsReason(reasons, NodeReasonStaleProbe) {
		t.Fatalf("expected %s, got %v", NodeReasonStaleProbe, reasons)
	}

	node.LastCheckedAt = now
	node.FreeBytes = 99
	reasons = evaluateNodeReasons(node, now, cacheTTL, true, minFree, hwm)
	if !containsReason(reasons, NodeReasonBelowMinFreeBytes) {
		t.Fatalf("expected %s, got %v", NodeReasonBelowMinFreeBytes, reasons)
	}

	node.FreeBytes = 200
	node.StoragePercent = 90
	reasons = evaluateNodeReasons(node, now, cacheTTL, true, minFree, hwm)
	if !containsReason(reasons, NodeReasonHighWatermarkReached) {
		t.Fatalf("expected %s, got %v", NodeReasonHighWatermarkReached, reasons)
	}
}

func TestChooseUploadNodeFromNodesMatchesEvaluationView(t *testing.T) {
	now := time.Now()
	cacheTTL := 15 * time.Second
	minFree := uint64(100)
	hwm := 85.0
	nodes := []FileserverNode{
		{ID: "fs-a", Weight: testNodeWeight(), Healthy: true, LastCheckedAt: now, FreeBytes: 500, StoragePercent: 40},
		{ID: "fs-b", Weight: testNodeWeight(), Healthy: true, LastCheckedAt: now, FreeBytes: 800, StoragePercent: 70},
		{ID: "fs-c", Weight: testNodeWeight(), Healthy: false, LastCheckedAt: now, FreeBytes: 900, StoragePercent: 10},
	}
	best, err := chooseUploadNodeFromNodes(nodes, now, cacheTTL, true, minFree, hwm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if best == nil || best.ID != "fs-b" {
		t.Fatalf("expected fs-b, got %+v", best)
	}

	origNodes := gateway.nodes
	origProbe := gateway.probeInterval
	origTTL := gateway.nodeCacheTTL
	origMin := gateway.minFreeBytes
	origHwm := gateway.highWatermarkPercent
	gateway.mu.Lock()
	gateway.nodes = nodes
	gateway.probeInterval = 10 * time.Second
	gateway.nodeCacheTTL = cacheTTL
	gateway.minFreeBytes = minFree
	gateway.highWatermarkPercent = hwm
	gateway.mu.Unlock()
	defer func() {
		gateway.mu.Lock()
		gateway.nodes = origNodes
		gateway.probeInterval = origProbe
		gateway.nodeCacheTTL = origTTL
		gateway.minFreeBytes = origMin
		gateway.highWatermarkPercent = origHwm
		gateway.mu.Unlock()
	}()

	view := FileserverNodeEvaluationView()
	if view.RecommendedNodeID != "fs-b" {
		t.Fatalf("expected recommended fs-b, got %s", view.RecommendedNodeID)
	}
}

func TestChooseUploadNodeFromNodesRejectsHighWatermark(t *testing.T) {
	now := time.Now()
	nodes := []FileserverNode{
		{ID: "fs-a", Weight: testNodeWeight(), Healthy: true, LastCheckedAt: now, FreeBytes: 900, StoragePercent: 85},
		{ID: "fs-b", Weight: testNodeWeight(), Healthy: true, LastCheckedAt: now, FreeBytes: 500, StoragePercent: 84.99},
	}
	best, err := chooseUploadNodeFromNodes(nodes, now, 15*time.Second, true, 100, 85)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if best == nil || best.ID != "fs-b" {
		t.Fatalf("expected fs-b below high watermark, got %+v", best)
	}
}

func TestChooseUploadNodeFromNodesNoCandidate(t *testing.T) {
	now := time.Now()
	nodes := []FileserverNode{
		{ID: "fs-a", Weight: testNodeWeight(), Healthy: false, LastCheckedAt: now, FreeBytes: 900, StoragePercent: 10},
		{ID: "fs-b", Weight: testNodeWeight(), Healthy: true, LastCheckedAt: now, FreeBytes: 500, StoragePercent: 85},
		{ID: "fs-c", Weight: testNodeWeight(), Healthy: true, LastCheckedAt: now, FreeBytes: 50, StoragePercent: 20},
	}
	if best, err := chooseUploadNodeFromNodes(nodes, now, 15*time.Second, true, 100, 85); err == nil || best != nil {
		t.Fatalf("expected no candidate, got best=%+v err=%v", best, err)
	}
}

func TestChooseUploadNodeFromNodesAllowsLowSpaceWhenDiskProtectionDisabled(t *testing.T) {
	now := time.Now()
	total := uint64(23 * 1024 * 1024 * 1024)
	nodes := []FileserverNode{
		{ID: "fs-small", Weight: testNodeWeight(), Healthy: true, LastCheckedAt: now, TotalBytes: total, FreeBytes: 4 * 1024 * 1024 * 1024, StoragePercent: 81.5},
	}
	best, err := chooseUploadNodeFromNodes(nodes, now, 15*time.Second, false, 20*1024*1024*1024, 85)
	if err != nil {
		t.Fatalf("expected low-space node to be accepted when disk protection is disabled, got %v", err)
	}
	if best == nil || best.ID != "fs-small" {
		t.Fatalf("expected fs-small, got %+v", best)
	}
	if reasons := evaluateNodeReasons(nodes[0], now, 15*time.Second, false, 20*1024*1024*1024, 85); len(reasons) != 0 {
		t.Fatalf("did not expect disk reasons when protection is disabled, got %v", reasons)
	}
}

func TestChooseUploadNodeFromNodesStillRejectsUnhealthyWhenDiskProtectionDisabled(t *testing.T) {
	now := time.Now()
	total := uint64(23 * 1024 * 1024 * 1024)
	nodes := []FileserverNode{
		{ID: "fs-small", Weight: testNodeWeight(), Healthy: false, LastCheckedAt: now, TotalBytes: total, FreeBytes: 4 * 1024 * 1024 * 1024, StoragePercent: 81.5},
	}
	if best, err := chooseUploadNodeFromNodes(nodes, now, 15*time.Second, false, 20*1024*1024*1024, 85); err == nil || best != nil {
		t.Fatalf("expected unhealthy node to be rejected, got best=%+v err=%v", best, err)
	}
}

func TestParseNodeListDefaultsWeightAndCleansWeightQuery(t *testing.T) {
	nodes := parseNodeList("10.0.0.11:18080,https://10.0.0.12:18080?weight=0,http://10.0.0.13:18080/path?weight=50&zone=a")
	if len(nodes) != 3 {
		t.Fatalf("node count = %d, want 3: %+v", len(nodes), nodes)
	}
	if nodes[0].Weight != DefaultFileserverNodeWeight {
		t.Fatalf("default weight = %d, want %d", nodes[0].Weight, DefaultFileserverNodeWeight)
	}
	if nodes[0].BaseURL != "http://10.0.0.11:18080" {
		t.Fatalf("default base url = %q", nodes[0].BaseURL)
	}
	if nodes[1].Weight != 0 {
		t.Fatalf("weight = %d, want 0", nodes[1].Weight)
	}
	if nodes[1].BaseURL != "https://10.0.0.12:18080" {
		t.Fatalf("weight query should be removed, got %q", nodes[1].BaseURL)
	}
	if nodes[2].Weight != 50 {
		t.Fatalf("weight = %d, want 50", nodes[2].Weight)
	}
	if nodes[2].BaseURL != "http://10.0.0.13:18080/path?zone=a" {
		t.Fatalf("non-weight query should remain, got %q", nodes[2].BaseURL)
	}
}

func TestParseFileserverNodeWeightInvalidOrNegativeIsZero(t *testing.T) {
	for _, value := range []string{"bad", "-1"} {
		if got := parseFileserverNodeWeight(value); got != 0 {
			t.Fatalf("parseFileserverNodeWeight(%q) = %d, want 0", value, got)
		}
	}
	if got := parseFileserverNodeWeight(""); got != DefaultFileserverNodeWeight {
		t.Fatalf("empty weight = %d, want default", got)
	}
	if got := parseFileserverNodeWeight("200"); got != 200 {
		t.Fatalf("weight = %d, want 200", got)
	}
}

func TestChooseUploadNodeFromNodesPrefersHigherWeight(t *testing.T) {
	now := time.Now()
	nodes := []FileserverNode{
		{ID: "fs-low-space-high-weight", Weight: 200, Healthy: true, LastCheckedAt: now, FreeBytes: 100, StoragePercent: 80},
		{ID: "fs-more-space-low-weight", Weight: 50, Healthy: true, LastCheckedAt: now, FreeBytes: 900, StoragePercent: 10},
	}
	best, err := chooseUploadNodeFromNodes(nodes, now, 15*time.Second, true, 1, 95)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if best == nil || best.ID != "fs-low-space-high-weight" {
		t.Fatalf("expected high weight node, got %+v", best)
	}
}

func TestChooseUploadNodeFromNodesRejectsZeroWeight(t *testing.T) {
	now := time.Now()
	zeroWeight := FileserverNode{ID: "fs-zero", Weight: 0, Healthy: true, LastCheckedAt: now, FreeBytes: 900, StoragePercent: 10}
	reasons := evaluateNodeReasons(zeroWeight, now, 15*time.Second, true, 1, 95)
	if !containsReason(reasons, NodeReasonUploadWeightZero) {
		t.Fatalf("expected %s, got %v", NodeReasonUploadWeightZero, reasons)
	}
	nodes := []FileserverNode{
		zeroWeight,
		{ID: "fs-active", Weight: 10, Healthy: true, LastCheckedAt: now, FreeBytes: 100, StoragePercent: 80},
	}
	best, err := chooseUploadNodeFromNodes(nodes, now, 15*time.Second, true, 1, 95)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if best == nil || best.ID != "fs-active" {
		t.Fatalf("expected active node, got %+v", best)
	}
}

func TestChooseUploadNodeFromNodesRejectsAllZeroWeight(t *testing.T) {
	now := time.Now()
	nodes := []FileserverNode{
		{ID: "fs-a", Weight: 0, Healthy: true, LastCheckedAt: now, FreeBytes: 900, StoragePercent: 10},
		{ID: "fs-b", Weight: 0, Healthy: true, LastCheckedAt: now, FreeBytes: 800, StoragePercent: 20},
	}
	if best, err := chooseUploadNodeFromNodes(nodes, now, 15*time.Second, true, 1, 95); err == nil || best != nil {
		t.Fatalf("expected no candidate, got best=%+v err=%v", best, err)
	}
}

func TestChooseUploadNodeRealtimeRefreshesProbe(t *testing.T) {
	var servers []*httptest.Server
	defer func() {
		for _, server := range servers {
			server.Close()
		}
	}()
	newServer := func(nodeID string, total uint64, free uint64) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/internal/v1/health":
				_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"status":"ok","node_id":"` + nodeID + `"}}`))
			case "/internal/v1/space":
				used := total - free
				_, _ = w.Write([]byte(fmt.Sprintf(`{"code":200,"message":"success","data":{"total_bytes":%d,"free_bytes":%d,"used_bytes":%d}}`, total, free, used)))
			default:
				http.NotFound(w, r)
			}
		}))
	}
	servers = append(servers, newServer("node-a", 1000, 100), newServer("node-b", 1000, 700))

	origNodes := gateway.nodes
	origMin := gateway.minFreeBytes
	origHwm := gateway.highWatermarkPercent
	origClient := gateway.httpClient
	gateway.mu.Lock()
	gateway.nodes = []FileserverNode{
		{ID: "old-a", BaseURL: servers[0].URL, Weight: testNodeWeight()},
		{ID: "old-b", BaseURL: servers[1].URL, Weight: testNodeWeight()},
	}
	gateway.minFreeBytes = 0
	gateway.highWatermarkPercent = 85
	gateway.httpClient = servers[0].Client()
	gateway.mu.Unlock()
	defer func() {
		gateway.mu.Lock()
		gateway.nodes = origNodes
		gateway.minFreeBytes = origMin
		gateway.highWatermarkPercent = origHwm
		gateway.httpClient = origClient
		gateway.mu.Unlock()
	}()

	best, err := ChooseUploadNodeRealtime()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if best == nil || best.ID != "node-b" {
		t.Fatalf("expected realtime probe to select node-b, got %+v", best)
	}
}

func TestProxyUploadUsesUploadHTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/files/upload" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("X-Uploader"); got != "admin" {
			t.Fatalf("X-Uploader = %q, want admin", got)
		}
		_, _ = w.Write([]byte(`{"code":200,"data":{"file_id":"fw-1"}}`))
	}))
	defer server.Close()

	origHTTPClient := gateway.httpClient
	origUploadHTTPClient := gateway.uploadHTTPClient
	gateway.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("ProxyUpload used the regular fileserver HTTP client")
		return nil, nil
	})}
	gateway.uploadHTTPClient = server.Client()
	defer func() {
		gateway.httpClient = origHTTPClient
		gateway.uploadHTTPClient = origUploadHTTPClient
	}()

	status, payload, err := ProxyUpload(
		FileserverNode{ID: "fs-test", BaseURL: server.URL},
		0,
		"admin",
		bytes.NewReader([]byte("payload")),
		"text/plain",
		int64(len("payload")),
	)
	if err != nil {
		t.Fatalf("ProxyUpload() error = %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200; payload=%s", status, payload)
	}
}

func TestFileserverGatewayDefaultUploadTimeout(t *testing.T) {
	if gateway.uploadHTTPClient == nil {
		t.Fatal("uploadHTTPClient is nil")
	}
	if gateway.uploadHTTPClient.Timeout != 5*time.Minute {
		t.Fatalf("upload timeout = %v, want 5m", gateway.uploadHTTPClient.Timeout)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func testNodeWeight() int {
	return DefaultFileserverNodeWeight
}

func containsReason(reasons []string, target string) bool {
	for _, reason := range reasons {
		if reason == target {
			return true
		}
	}
	return false
}
