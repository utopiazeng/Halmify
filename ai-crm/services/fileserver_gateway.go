package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/server/web"
)

type FileserverNode struct {
	ID             string
	BaseURL        string
	Weight         int
	Healthy        bool
	LatencyMS      float64
	TotalBytes     uint64
	FreeBytes      uint64
	UsedBytes      uint64
	LastCheckedAt  time.Time
	LastError      string
	StoragePercent float64
}

const (
	NodeReasonUnhealthy            = "unhealthy"
	NodeReasonStaleProbe           = "stale_probe"
	NodeReasonBelowMinFreeBytes    = "below_min_free_bytes"
	NodeReasonHighWatermarkReached = "high_watermark_reached"
	NodeReasonUploadWeightZero     = "upload_weight_zero"
	DefaultFileserverNodeWeight    = 100
)

type FileserverThresholdSnapshot struct {
	ProbeIntervalSec      int64   `json:"probe_interval_sec"`
	CacheTTLSec           int64   `json:"cache_ttl_sec"`
	DiskProtectionEnabled bool    `json:"disk_protection_enabled"`
	MinFreeBytes          uint64  `json:"min_free_bytes"`
	HighWatermarkPercent  float64 `json:"high_watermark_percent"`
}

type FileserverNodeEvaluation struct {
	ID             string    `json:"id"`
	BaseURL        string    `json:"base_url"`
	Weight         int       `json:"weight"`
	Healthy        bool      `json:"healthy"`
	LastCheckedAt  time.Time `json:"last_checked_at"`
	LastError      string    `json:"last_error"`
	LatencyMS      float64   `json:"latency_ms"`
	TotalBytes     uint64    `json:"total_bytes"`
	FreeBytes      uint64    `json:"free_bytes"`
	UsedBytes      uint64    `json:"used_bytes"`
	StoragePercent float64   `json:"storage_percent"`
	IsCandidate    bool      `json:"is_candidate"`
	Reasons        []string  `json:"reasons"`
}

type FileserverNodeEvaluationSnapshot struct {
	Thresholds        FileserverThresholdSnapshot `json:"thresholds"`
	RecommendedNodeID string                      `json:"recommended_node_id"`
	Nodes             []FileserverNodeEvaluation  `json:"nodes"`
}

type FileserverTenantUsage struct {
	UsedBytes     uint64 `json:"used_bytes"`
	PhysicalCount int64  `json:"physical_count"`
	RecordCount   int64  `json:"record_count"`
}

type fileserverGateway struct {
	mu                    sync.RWMutex
	nodes                 []FileserverNode
	appID                 string
	appSecret             string
	probeInterval         time.Duration
	nodeCacheTTL          time.Duration
	diskProtectionEnabled bool
	minFreeBytes          uint64
	highWatermarkPercent  float64
	httpClient            *http.Client
	uploadHTTPClient      *http.Client
}

var gateway = &fileserverGateway{
	nodes:                 []FileserverNode{},
	probeInterval:         10 * time.Second,
	nodeCacheTTL:          15 * time.Second,
	diskProtectionEnabled: true,
	minFreeBytes:          20 * 1024 * 1024 * 1024,
	highWatermarkPercent:  85,
	httpClient:            &http.Client{Timeout: 30 * time.Second},
	uploadHTTPClient:      &http.Client{Timeout: 5 * time.Minute},
}

func InitFileserverGateway() {
	gateway.loadConfig()
}

func StartFileserverProbeScheduler() {
	withJobLock("fileserver_probe_scheduler", gateway.probeAllNodes)
	go func() {
		ticker := time.NewTicker(gateway.probeInterval)
		defer ticker.Stop()
		for range ticker.C {
			withJobLock("fileserver_probe_scheduler", gateway.probeAllNodes)
		}
	}()
}

func FileserverNodesSnapshot() []FileserverNode {
	gateway.mu.RLock()
	defer gateway.mu.RUnlock()
	result := make([]FileserverNode, len(gateway.nodes))
	copy(result, gateway.nodes)
	return result
}

func ChooseUploadNode() (*FileserverNode, error) {
	nodes := FileserverNodesSnapshot()
	cacheTTL, diskProtectionEnabled, minFreeBytes, highWatermark := gateway.routingThresholds()
	return chooseUploadNodeFromNodes(nodes, time.Now(), cacheTTL, diskProtectionEnabled, minFreeBytes, highWatermark)
}

func ChooseUploadNodeRealtime() (*FileserverNode, error) {
	nodes := gateway.probeAllNodesNow()
	_, diskProtectionEnabled, minFreeBytes, highWatermark := gateway.routingThresholds()
	return chooseUploadNodeFromNodes(nodes, time.Now(), 0, diskProtectionEnabled, minFreeBytes, highWatermark)
}

func FileserverNodeByID(nodeID string) (FileserverNode, bool) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return FileserverNode{}, false
	}
	nodes := FileserverNodesSnapshot()
	for _, node := range nodes {
		if node.ID == nodeID {
			return node, true
		}
	}
	return FileserverNode{}, false
}

func FileserverNodeEvaluationView() FileserverNodeEvaluationSnapshot {
	nodes := FileserverNodesSnapshot()
	probeInterval, cacheTTL, diskProtectionEnabled, minFreeBytes, highWatermark := gateway.thresholdSnapshot()
	now := time.Now()
	items := make([]FileserverNodeEvaluation, 0, len(nodes))
	for _, node := range nodes {
		reasons := evaluateNodeReasons(node, now, cacheTTL, diskProtectionEnabled, minFreeBytes, highWatermark)
		items = append(items, FileserverNodeEvaluation{
			ID:             node.ID,
			BaseURL:        node.BaseURL,
			Weight:         node.Weight,
			Healthy:        node.Healthy,
			LastCheckedAt:  node.LastCheckedAt,
			LastError:      node.LastError,
			LatencyMS:      node.LatencyMS,
			TotalBytes:     node.TotalBytes,
			FreeBytes:      node.FreeBytes,
			UsedBytes:      node.UsedBytes,
			StoragePercent: node.StoragePercent,
			IsCandidate:    len(reasons) == 0,
			Reasons:        reasons,
		})
	}
	recommendedNode := ""
	if best, err := chooseUploadNodeFromNodes(nodes, now, cacheTTL, diskProtectionEnabled, minFreeBytes, highWatermark); err == nil && best != nil {
		recommendedNode = best.ID
	}
	return FileserverNodeEvaluationSnapshot{
		Thresholds: FileserverThresholdSnapshot{
			ProbeIntervalSec:      int64(probeInterval / time.Second),
			CacheTTLSec:           int64(cacheTTL / time.Second),
			DiskProtectionEnabled: diskProtectionEnabled,
			MinFreeBytes:          minFreeBytes,
			HighWatermarkPercent:  highWatermark,
		},
		RecommendedNodeID: recommendedNode,
		Nodes:             items,
	}
}

func PostInternalJSON(node FileserverNode, path string, tenantID uint, body []byte) (int, []byte, error) {
	endpoint := strings.TrimRight(node.BaseURL, "/") + path
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	applyInternalHeaders(req, tenantID, body)
	resp, err := gateway.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, payload, nil
}

func ProxyUpload(node FileserverNode, tenantID uint, uploader string, reqBody io.Reader, contentType string, contentLength int64) (int, []byte, error) {
	endpoint := strings.TrimRight(node.BaseURL, "/") + "/internal/v1/files/upload"
	req, err := http.NewRequest(http.MethodPost, endpoint, reqBody)
	if err != nil {
		return 0, nil, err
	}
	if contentLength > 0 {
		req.ContentLength = contentLength
	}
	if contentType == "" {
		contentType = "multipart/form-data"
	}
	req.Header.Set("Content-Type", contentType)
	if strings.TrimSpace(uploader) != "" {
		req.Header.Set("X-Uploader", uploader)
	}
	applyInternalHeaders(req, tenantID, nil)
	resp, err := gateway.uploadHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, payload, nil
}

func GetInternal(node FileserverNode, path string, rawQuery string, tenantID uint, rangeHeader string) (*http.Response, error) {
	endpoint := strings.TrimRight(node.BaseURL, "/") + path
	if rawQuery != "" {
		endpoint += "?" + rawQuery
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(rangeHeader) != "" {
		req.Header.Set("Range", rangeHeader)
	}
	applyInternalHeaders(req, tenantID, nil)
	return gateway.httpClient.Do(req)
}

func GetInternalJSON(node FileserverNode, path string, rawQuery string, tenantID uint) (int, []byte, error) {
	resp, err := GetInternal(node, path, rawQuery, tenantID, "")
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, payload, nil
}

func FileserverTenantUsedBytes(tenantID uint) (uint64, error) {
	nodes := FileserverNodesSnapshot()
	var total uint64
	var queried int
	failures := make([]string, 0, len(nodes))
	for _, node := range nodes {
		status, payload, err := GetInternalJSON(node, "/internal/v1/files/tenant/usage", "", tenantID)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s tenant usage error: %v", node.BaseURL, err))
			continue
		}
		if status < 200 || status >= 300 {
			failures = append(failures, fmt.Sprintf("%s tenant usage http %d: %s", node.BaseURL, status, payloadSnippet(payload)))
			continue
		}
		var envelope struct {
			Data FileserverTenantUsage `json:"data"`
		}
		if err := json.Unmarshal(payload, &envelope); err != nil {
			failures = append(failures, fmt.Sprintf("%s tenant usage decode: %v", node.BaseURL, err))
			continue
		}
		total += envelope.Data.UsedBytes
		queried++
	}
	if queried == 0 {
		if len(failures) > 0 {
			return 0, fmt.Errorf("no reachable fileserver tenant usage endpoint: %s", strings.Join(failures, "; "))
		}
		return 0, errors.New("no reachable fileserver tenant usage endpoint: no fileserver nodes configured")
	}
	return total, nil
}

func DeleteInternal(node FileserverNode, path string, tenantID uint) (int, []byte, error) {
	endpoint := strings.TrimRight(node.BaseURL, "/") + path
	req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return 0, nil, err
	}
	applyInternalHeaders(req, tenantID, nil)
	resp, err := gateway.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, payload, nil
}

func applyInternalHeaders(req *http.Request, tenantID uint, body []byte) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	req.Header.Set("X-App-Id", gateway.appID)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Tenant-ID", strconv.FormatUint(uint64(tenantID), 10))
	query := ""
	if req.URL != nil {
		query = req.URL.RawQuery
	}
	bodyHash := sha256Hex(body)
	if strings.HasPrefix(strings.ToLower(req.Header.Get("Content-Type")), "multipart/form-data") {
		bodyHash = sha256Hex(nil)
	}
	canonical := strings.Join([]string{
		req.Method,
		req.URL.Path,
		query,
		bodyHash,
		ts,
		nonce,
		req.Header.Get("X-Tenant-ID"),
	}, "\n")
	sig := hmacHex(gateway.appSecret, canonical)
	req.Header.Set("X-Signature", sig)
}

func hmacHex(secret, raw string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(raw))
	return hex.EncodeToString(mac.Sum(nil))
}

func sha256Hex(body []byte) string {
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}

func (g *fileserverGateway) loadConfig() {
	nodesValue, _ := web.AppConfig.String("fileserver_nodes")
	if env := os.Getenv("FILESERVER_NODES"); strings.TrimSpace(env) != "" {
		nodesValue = env
	}
	appID, _ := web.AppConfig.String("fileserver_app_id")
	if env := os.Getenv("FILESERVER_APP_ID"); strings.TrimSpace(env) != "" {
		appID = env
	}
	appSecret, _ := web.AppConfig.String("fileserver_app_secret")
	if env := os.Getenv("FILESERVER_APP_SECRET"); strings.TrimSpace(env) != "" {
		appSecret = env
	}
	probeSec, _ := web.AppConfig.Int("fileserver_probe_interval_sec")
	if env := os.Getenv("FILESERVER_PROBE_INTERVAL_SEC"); strings.TrimSpace(env) != "" {
		if parsed, err := strconv.Atoi(env); err == nil {
			probeSec = parsed
		}
	}
	cacheSec, _ := web.AppConfig.Int("fileserver_cache_ttl_sec")
	if env := os.Getenv("FILESERVER_CACHE_TTL_SEC"); strings.TrimSpace(env) != "" {
		if parsed, err := strconv.Atoi(env); err == nil {
			cacheSec = parsed
		}
	}
	diskProtectionEnabled := web.AppConfig.DefaultBool("fileserver_disk_protection_enabled", true)
	if env := os.Getenv("FILESERVER_DISK_PROTECTION_ENABLED"); strings.TrimSpace(env) != "" {
		if parsed, err := strconv.ParseBool(env); err == nil {
			diskProtectionEnabled = parsed
		}
	}
	minFree, _ := web.AppConfig.Int64("fileserver_min_free_bytes")
	if env := os.Getenv("FILESERVER_MIN_FREE_BYTES"); strings.TrimSpace(env) != "" {
		if parsed, err := strconv.ParseInt(env, 10, 64); err == nil {
			minFree = parsed
		}
	}
	hwm, _ := web.AppConfig.Float("fileserver_high_watermark_percent")
	if env := os.Getenv("FILESERVER_HIGH_WATERMARK_PERCENT"); strings.TrimSpace(env) != "" {
		if parsed, err := strconv.ParseFloat(env, 64); err == nil {
			hwm = parsed
		}
	}
	uploadTimeoutSec := web.AppConfig.DefaultInt("fileserver_upload_timeout_sec", 300)
	if env := os.Getenv("FILESERVER_UPLOAD_TIMEOUT_SEC"); strings.TrimSpace(env) != "" {
		if parsed, err := strconv.Atoi(env); err == nil {
			uploadTimeoutSec = parsed
		}
	}
	if probeSec <= 0 {
		probeSec = 10
	}
	if cacheSec <= 0 {
		cacheSec = 15
	}
	if minFree <= 0 {
		minFree = 20 * 1024 * 1024 * 1024
	}
	if hwm <= 0 || hwm >= 100 {
		hwm = 85
	}
	if uploadTimeoutSec <= 0 {
		uploadTimeoutSec = 300
	}
	if appID == "" {
		appID = "ai-crm"
	}
	if appSecret == "" {
		appSecret = "change-me"
	}

	parsedNodes := parseNodeList(nodesValue)
	g.mu.Lock()
	g.nodes = parsedNodes
	g.appID = appID
	g.appSecret = appSecret
	g.probeInterval = time.Duration(probeSec) * time.Second
	g.nodeCacheTTL = time.Duration(cacheSec) * time.Second
	g.diskProtectionEnabled = diskProtectionEnabled
	g.minFreeBytes = uint64(minFree)
	g.highWatermarkPercent = hwm
	g.uploadHTTPClient = &http.Client{Timeout: time.Duration(uploadTimeoutSec) * time.Second}
	g.mu.Unlock()
}

func parseNodeList(raw string) []FileserverNode {
	items := make([]FileserverNode, 0)
	for idx, part := range strings.Split(raw, ",") {
		base := strings.TrimSpace(part)
		if base == "" {
			continue
		}
		if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
			base = "http://" + base
		}
		parsedURL, err := url.Parse(base)
		if err != nil {
			continue
		}
		weight := parseFileserverNodeWeight(parsedURL.Query().Get("weight"))
		query := parsedURL.Query()
		query.Del("weight")
		parsedURL.RawQuery = query.Encode()
		items = append(items, FileserverNode{
			ID:      fmt.Sprintf("fs-%d", idx+1),
			BaseURL: strings.TrimRight(parsedURL.String(), "/"),
			Weight:  weight,
		})
	}
	return items
}

func parseFileserverNodeWeight(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultFileserverNodeWeight
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

type probeHealthResponse struct {
	Code int `json:"code"`
	Data struct {
		Status string `json:"status"`
		NodeID string `json:"node_id"`
	} `json:"data"`
}

type probeSpaceResponse struct {
	Code int `json:"code"`
	Data struct {
		TotalBytes uint64 `json:"total_bytes"`
		FreeBytes  uint64 `json:"free_bytes"`
		UsedBytes  uint64 `json:"used_bytes"`
	} `json:"data"`
}

func (g *fileserverGateway) probeAllNodes() {
	_ = g.probeAllNodesNow()
}

func (g *fileserverGateway) probeAllNodesNow() []FileserverNode {
	g.mu.RLock()
	nodes := make([]FileserverNode, len(g.nodes))
	copy(nodes, g.nodes)
	g.mu.RUnlock()
	if len(nodes) == 0 {
		return nodes
	}
	for i := range nodes {
		n := &nodes[i]
		now := time.Now()
		start := time.Now()
		health, healthErr := g.probeHealth(*n)
		n.LatencyMS = roundMilliseconds(time.Since(start))
		if healthErr != nil {
			n.Healthy = false
			n.LastError = healthErr.Error()
			n.LastCheckedAt = now
			continue
		}
		if health.NodeID != "" {
			n.ID = health.NodeID
		}
		if strings.ToLower(health.Status) != "ok" {
			n.Healthy = false
			n.LastError = "health not ok"
			n.LastCheckedAt = now
			continue
		}
		space, err := g.probeSpace(*n)
		n.LatencyMS = roundMilliseconds(time.Since(start))
		if err != nil {
			n.Healthy = false
			n.LastError = err.Error()
			n.LastCheckedAt = now
			continue
		}
		n.Healthy = true
		n.TotalBytes = space.Data.TotalBytes
		n.FreeBytes = space.Data.FreeBytes
		n.UsedBytes = space.Data.UsedBytes
		n.LastError = ""
		n.LastCheckedAt = now
		if n.TotalBytes > 0 {
			n.StoragePercent = (float64(n.UsedBytes) / float64(n.TotalBytes)) * 100
		}
	}
	g.mu.Lock()
	g.nodes = nodes
	g.mu.Unlock()
	return nodes
}

type probeHealthResult struct {
	Status string
	NodeID string
}

func (g *fileserverGateway) probeHealth(node FileserverNode) (probeHealthResult, error) {
	resp, err := g.GetSigned(http.MethodGet, strings.TrimRight(node.BaseURL, "/")+"/internal/v1/health", nil)
	if err != nil {
		return probeHealthResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return probeHealthResult{}, fmt.Errorf("health http %d: %s", resp.StatusCode, readResponseSnippet(resp))
	}
	var parsed probeHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return probeHealthResult{}, err
	}
	if parsed.Data.Status == "" {
		return probeHealthResult{}, errors.New("empty health status")
	}
	return probeHealthResult{Status: parsed.Data.Status, NodeID: strings.TrimSpace(parsed.Data.NodeID)}, nil
}

func (g *fileserverGateway) probeSpace(node FileserverNode) (probeSpaceResponse, error) {
	resp, err := g.GetSigned(http.MethodGet, strings.TrimRight(node.BaseURL, "/")+"/internal/v1/space", nil)
	if err != nil {
		return probeSpaceResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return probeSpaceResponse{}, fmt.Errorf("space http %d: %s", resp.StatusCode, readResponseSnippet(resp))
	}
	var parsed probeSpaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return probeSpaceResponse{}, err
	}
	return parsed, nil
}

func readResponseSnippet(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return payloadSnippet(payload)
}

func payloadSnippet(payload []byte) string {
	text := strings.TrimSpace(string(payload))
	if len(text) > 512 {
		return text[:512]
	}
	return text
}

func (g *fileserverGateway) GetSigned(method, endpoint string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, endpoint, bodyReader)
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	applyInternalHeaders(req, 0, body)
	return g.httpClient.Do(req)
}

func chooseUploadNodeFromNodes(nodes []FileserverNode, now time.Time, cacheTTL time.Duration, diskProtectionEnabled bool, minFreeBytes uint64, highWatermark float64) (*FileserverNode, error) {
	candidates := make([]FileserverNode, 0, len(nodes))
	for _, node := range nodes {
		if len(evaluateNodeReasons(node, now, cacheTTL, diskProtectionEnabled, minFreeBytes, highWatermark)) == 0 {
			candidates = append(candidates, node)
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("no fileserver node available")
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Weight != candidates[j].Weight {
			return candidates[i].Weight > candidates[j].Weight
		}
		if candidates[i].FreeBytes == candidates[j].FreeBytes {
			return candidates[i].StoragePercent < candidates[j].StoragePercent
		}
		return candidates[i].FreeBytes > candidates[j].FreeBytes
	})
	best := candidates[0]
	return &best, nil
}

func evaluateNodeReasons(node FileserverNode, now time.Time, cacheTTL time.Duration, diskProtectionEnabled bool, minFreeBytes uint64, highWatermark float64) []string {
	reasons := make([]string, 0, 4)
	if !node.Healthy {
		reasons = append(reasons, NodeReasonUnhealthy)
	}
	if node.Weight <= 0 {
		reasons = append(reasons, NodeReasonUploadWeightZero)
	}
	if cacheTTL > 0 && (node.LastCheckedAt.IsZero() || now.Sub(node.LastCheckedAt) > cacheTTL) {
		reasons = append(reasons, NodeReasonStaleProbe)
	}
	if diskProtectionEnabled {
		if node.FreeBytes < minFreeBytes {
			reasons = append(reasons, NodeReasonBelowMinFreeBytes)
		}
		if node.StoragePercent >= highWatermark {
			reasons = append(reasons, NodeReasonHighWatermarkReached)
		}
	}
	return reasons
}

func roundMilliseconds(duration time.Duration) float64 {
	value, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", float64(duration.Microseconds())/1000), 64)
	return value
}

func (g *fileserverGateway) routingThresholds() (time.Duration, bool, uint64, float64) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.nodeCacheTTL, g.diskProtectionEnabled, g.minFreeBytes, g.highWatermarkPercent
}

func (g *fileserverGateway) thresholdSnapshot() (time.Duration, time.Duration, bool, uint64, float64) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.probeInterval, g.nodeCacheTTL, g.diskProtectionEnabled, g.minFreeBytes, g.highWatermarkPercent
}
