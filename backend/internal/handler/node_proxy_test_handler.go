package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"server-sing-box-2/backend/internal/domain"

	"github.com/gin-gonic/gin"
)

const (
	defaultProxyTestURL     = "https://cp.cloudflare.com/generate_204"
	defaultProxyTestTimeout = 5 * time.Second
)

type nodeProxyTestResult struct {
	NodeID      uint   `json:"nodeId"`
	NodeName    string `json:"nodeName"`
	Status      string `json:"status"`
	LatencyMs   *int   `json:"latencyMs,omitempty"`
	ExitIP      string `json:"exitIp,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
	Error       string `json:"error,omitempty"`
	TestURL     string `json:"testUrl"`
	CheckedAt   string `json:"checkedAt"`
}

type mihomoDelayResponse struct {
	Delay int `json:"delay"`
}

type ipGeoResponse struct {
	Status      string `json:"status"`
	Query       string `json:"query"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	Message     string `json:"message"`
}

func (h *Handler) TestNodeProxy(c *gin.Context) {
	node, ok := h.findOwnedNode(c)
	if !ok {
		return
	}
	if !canTestNodeProxy(node) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node is not ready for proxy test"})
		return
	}
	result := h.testNodeProxy(c.Request.Context(), node)
	c.JSON(http.StatusOK, result)
}

func (h *Handler) TestAllNodeProxies(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var nodes []domain.ProtocolNode
	if err := h.db.Where("user_id = ? AND status IN ?", userID, []domain.NodeStatus{domain.NodeStatusImported, domain.NodeStatusInstallOK}).Order("created_at DESC").Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list nodes failed"})
		return
	}

	results := make([]nodeProxyTestResult, len(nodes))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 2)
	for index, node := range nodes {
		wg.Add(1)
		go func(index int, node domain.ProtocolNode) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[index] = h.testNodeProxy(c.Request.Context(), node)
		}(index, node)
	}
	wg.Wait()

	c.JSON(http.StatusOK, results)
}

func canTestNodeProxy(node domain.ProtocolNode) bool {
	return node.Status == domain.NodeStatusImported || node.Status == domain.NodeStatusInstallOK
}

func resolveMihomoBin() (string, bool) {
	candidates := []string{strings.TrimSpace(os.Getenv("MIHOMO_BIN")), "mihomo"}
	if workingDir, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(workingDir, "bin/mihomo"),
			filepath.Join(workingDir, "backend/bin/mihomo"),
		)
	}
	candidates = append(candidates, "/usr/local/bin/mihomo", "/opt/homebrew/bin/mihomo")

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved, true
		}
	}
	return "", false
}

func (h *Handler) testNodeProxy(parent context.Context, node domain.ProtocolNode) nodeProxyTestResult {
	result := nodeProxyTestResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Status:    "failed",
		TestURL:   proxyTestURL(),
		CheckedAt: time.Now().Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()

	view, dependencies, err := h.proxyTestNodeView(node)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	mihomoBin, ok := resolveMihomoBin()
	if !ok {
		result.Error = "mihomo binary not found"
		return result
	}

	controllerPort, err := freeLocalPort()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	mixedPort, err := freeLocalPort()
	if err != nil {
		result.Error = err.Error()
		return result
	}

	secret := randomHex(16)
	config := renderMihomoProxyTestConfig(controllerPort, mixedPort, secret, view, dependencies)
	tempDir, err := os.MkdirTemp("", "node-proxy-test-*")
	if err != nil {
		result.Error = "create temp dir failed"
		return result
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		result.Error = "write mihomo config failed"
		return result
	}

	var output bytes.Buffer
	cmd := exec.CommandContext(ctx, mihomoBin, "-d", tempDir, "-f", configPath)
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		result.Error = "start mihomo failed: " + err.Error()
		return result
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	if err := waitMihomoController(ctx, controllerPort, secret); err != nil {
		result.Error = err.Error()
		if logs := strings.TrimSpace(output.String()); logs != "" {
			result.Error += ": " + lastLogLine(logs)
		}
		return result
	}

	delay, err := testMihomoDelay(ctx, controllerPort, secret, view.Name, result.TestURL)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.LatencyMs = &delay
	result.Status = "ok"

	geo, err := lookupProxyExitGeo(ctx, mixedPort)
	if err == nil {
		result.ExitIP = geo.Query
		result.Country = geo.Country
		result.CountryCode = geo.CountryCode
	}

	return result
}

func (h *Handler) proxyTestNodeView(node domain.ProtocolNode) (subscriptionNodeView, []subscriptionNodeView, error) {
	byID := map[uint]domain.ProtocolNode{node.ID: node}
	if err := h.loadChainProxyDependencies(byID, []domain.ProtocolNode{node}); err != nil {
		return subscriptionNodeView{}, nil, err
	}
	nodes := protocolNodesFromMap(byID)
	sensitiveByNodeID, err := h.subscriptionNodeSensitiveValues(nodes)
	if err != nil {
		return subscriptionNodeView{}, nil, err
	}

	view := subscriptionNodeViewFromNode(node, byID, sensitiveByNodeID[node.ID])
	if strings.TrimSpace(view.Address) == "" || view.Port == 0 {
		return subscriptionNodeView{}, nil, errors.New("node endpoint is incomplete")
	}

	dependencies := make([]subscriptionNodeView, 0, len(nodes)-1)
	for _, item := range nodes {
		if item.ID == node.ID {
			continue
		}
		dependencies = append(dependencies, subscriptionNodeViewFromNode(item, byID, sensitiveByNodeID[item.ID]))
	}
	return view, dependencies, nil
}

func renderMihomoProxyTestConfig(controllerPort int, mixedPort int, secret string, target subscriptionNodeView, dependencies []subscriptionNodeView) string {
	lines := []string{
		fmt.Sprintf("mixed-port: %d", mixedPort),
		"allow-lan: false",
		"bind-address: 127.0.0.1",
		"mode: rule",
		"log-level: warning",
		"unified-delay: true",
		"tcp-concurrent: true",
		fmt.Sprintf("external-controller: 127.0.0.1:%d", controllerPort),
		"secret: " + yamlQuote(secret),
		"proxies:",
	}
	for _, dependency := range dependencies {
		lines = append(lines, dependency.clashProxyLines()...)
	}
	lines = append(lines, target.clashProxyLines()...)
	lines = append(lines,
		"proxy-groups:",
		"  - name: "+yamlQuote("ProxyTest"),
		"    type: select",
		"    proxies:",
		"      - "+yamlQuote(target.Name),
		"rules:",
		"  - MATCH,ProxyTest",
	)
	return strings.Join(lines, "\n") + "\n"
}

func waitMihomoController(ctx context.Context, port int, secret string) error {
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/version", port), nil)
		req.Header.Set("Authorization", "Bearer "+secret)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	return errors.New("mihomo controller did not become ready")
}

func proxyTestURL() string {
	if value := strings.TrimSpace(os.Getenv("PROXY_TEST_URL")); value != "" {
		return value
	}
	return defaultProxyTestURL
}

func testMihomoDelay(ctx context.Context, port int, secret string, proxyName string, testURL string) (int, error) {
	endpoint := fmt.Sprintf(
		"http://127.0.0.1:%d/proxies/%s/delay?timeout=%d&url=%s",
		port,
		url.PathEscape(proxyName),
		int(defaultProxyTestTimeout/time.Millisecond),
		url.QueryEscape(testURL),
	)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("mihomo delay test failed: %s", resp.Status)
	}
	var body mihomoDelayResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, err
	}
	if body.Delay <= 0 {
		return 0, errors.New("mihomo returned empty delay")
	}
	return body.Delay, nil
}

func lookupProxyExitGeo(ctx context.Context, mixedPort int) (ipGeoResponse, error) {
	proxyURL, _ := url.Parse("http://127.0.0.1:" + strconv.Itoa(mixedPort))
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://ip-api.com/json/?fields=status,message,country,countryCode,query", nil)
	resp, err := client.Do(req)
	if err != nil {
		return ipGeoResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return ipGeoResponse{}, fmt.Errorf("geo lookup failed: %s", resp.Status)
	}
	var geo ipGeoResponse
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return ipGeoResponse{}, err
	}
	if geo.Status != "" && geo.Status != "success" {
		if geo.Message != "" {
			return ipGeoResponse{}, errors.New(geo.Message)
		}
		return ipGeoResponse{}, errors.New("geo lookup failed")
	}
	return geo, nil
}

func freeLocalPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func randomHex(size int) string {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(data)
}

func lastLogLine(logs string) string {
	lines := strings.Split(logs, "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		if line := strings.TrimSpace(lines[index]); line != "" {
			return line
		}
	}
	return ""
}
