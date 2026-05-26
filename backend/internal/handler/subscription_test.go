package handler_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"server-sing-box-2/backend/internal/domain"
	"server-sing-box-2/backend/internal/security"
)

type subscriptionResponse struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	Format          string `json:"format"`
	ClashTemplate   string `json:"clashTemplate"`
	ClashTemplateID *uint  `json:"clashTemplateId"`
	NodeIDs         []uint `json:"nodeIds"`
	NodeCount       int    `json:"nodeCount"`
	Token           string `json:"token"`
	SubscriptionURL string `json:"subscriptionUrl"`
}

type clashTemplateResponse struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

func TestSubscriptionCRUDPublicAccessAndReset(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-user", "sub-user@example.com")
	node := importTestNode(t, app, token, "Sub Node")

	createBody := `{"name":"Main Subscription","format":"sing-box","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `],"remark":"main"}`
	createRes := performRequest(app, http.MethodPost, "/api/v1/subscriptions", createBody, token)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}

	var created subscriptionResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Token == "" || created.SubscriptionURL == "" || created.NodeCount != 1 {
		t.Fatalf("unexpected created subscription: %+v", created)
	}

	publicRes := performRequest(app, http.MethodGet, created.SubscriptionURL, "", "")
	if publicRes.Code != http.StatusOK {
		t.Fatalf("expected public status 200, got %d: %s", publicRes.Code, publicRes.Body.String())
	}
	if !strings.Contains(publicRes.Body.String(), "outbounds") {
		t.Fatalf("expected sing-box content, got %s", publicRes.Body.String())
	}

	updateBody := `{"name":"Disabled Subscription","format":"base64","enabled":false,"nodeIds":[` + strconvUint(node.ID) + `]}`
	updateRes := performRequest(app, http.MethodPut, "/api/v1/subscriptions/1", updateBody, token)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d: %s", updateRes.Code, updateRes.Body.String())
	}

	disabledPublicRes := performRequest(app, http.MethodGet, created.SubscriptionURL, "", "")
	if disabledPublicRes.Code != http.StatusForbidden {
		t.Fatalf("expected disabled public status 403, got %d", disabledPublicRes.Code)
	}

	enableBody := `{"name":"Enabled Subscription","format":"base64","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	if res := performRequest(app, http.MethodPut, "/api/v1/subscriptions/1", enableBody, token); res.Code != http.StatusOK {
		t.Fatalf("expected enable status 200, got %d", res.Code)
	}

	resetRes := performRequest(app, http.MethodPost, "/api/v1/subscriptions/1/reset-token", "", token)
	if resetRes.Code != http.StatusOK {
		t.Fatalf("expected reset status 200, got %d: %s", resetRes.Code, resetRes.Body.String())
	}

	var reset subscriptionResponse
	if err := json.Unmarshal(resetRes.Body.Bytes(), &reset); err != nil {
		t.Fatalf("decode reset response: %v", err)
	}
	if reset.Token == "" || reset.Token == created.Token {
		t.Fatalf("expected new token, got old=%q new=%q", created.Token, reset.Token)
	}
	if oldRes := performRequest(app, http.MethodGet, created.SubscriptionURL, "", ""); oldRes.Code != http.StatusNotFound {
		t.Fatalf("expected old token status 404, got %d", oldRes.Code)
	}
	if newRes := performRequest(app, http.MethodGet, reset.SubscriptionURL, "", ""); newRes.Code != http.StatusOK {
		t.Fatalf("expected new token status 200, got %d", newRes.Code)
	}

	deleteRes := performRequest(app, http.MethodDelete, "/api/v1/subscriptions/1", "", token)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("expected delete status 204, got %d", deleteRes.Code)
	}
	if deletedRes := performRequest(app, http.MethodGet, reset.SubscriptionURL, "", ""); deletedRes.Code != http.StatusNotFound {
		t.Fatalf("expected deleted token status 404, got %d", deletedRes.Code)
	}
}

func TestSubscriptionRejectsCrossUserNode(t *testing.T) {
	app := testRouter(t)
	ownerToken := registerTestUser(t, app, "sub-owner", "sub-owner@example.com")
	otherToken := registerTestUser(t, app, "sub-other", "sub-other@example.com")
	ownerNode := importTestNode(t, app, ownerToken, "Owner Node")

	body := `{"name":"Bad Subscription","format":"sing-box","enabled":true,"nodeIds":[` + strconvUint(ownerNode.ID) + `]}`
	res := performRequest(app, http.MethodPost, "/api/v1/subscriptions", body, otherToken)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected cross-user node status 400, got %d: %s", res.Code, res.Body.String())
	}
}

func TestSubscriptionRenderingKeepsRawLinkForBase64(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-raw", "sub-raw@example.com")
	rawLink := "hy2://example.com:8443#Imported-HY2"

	nodeBody := `{"mode":"link","rawLink":"` + rawLink + `"}`
	nodeRes := performRequest(app, http.MethodPost, "/api/v1/nodes/import", nodeBody, token)
	if nodeRes.Code != http.StatusCreated {
		t.Fatalf("expected import status 201, got %d: %s", nodeRes.Code, nodeRes.Body.String())
	}
	var node nodeResponse
	if err := json.Unmarshal(nodeRes.Body.Bytes(), &node); err != nil {
		t.Fatalf("decode node response: %v", err)
	}

	createBody := `{"name":"Raw Subscription","format":"base64","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	createRes := performRequest(app, http.MethodPost, "/api/v1/subscriptions", createBody, token)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}
	var subscription subscriptionResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &subscription); err != nil {
		t.Fatalf("decode subscription response: %v", err)
	}

	publicRes := performRequest(app, http.MethodGet, subscription.SubscriptionURL, "", "")
	if publicRes.Code != http.StatusOK {
		t.Fatalf("expected public status 200, got %d: %s", publicRes.Code, publicRes.Body.String())
	}
	decoded, err := base64.StdEncoding.DecodeString(publicRes.Body.String())
	if err != nil {
		t.Fatalf("decode base64 subscription: %v", err)
	}
	if string(decoded) != rawLink {
		t.Fatalf("expected raw link %q, got %q", rawLink, string(decoded))
	}
}

func TestSubscriptionRenderingUsesPublicPortAndClientShapes(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-render", "sub-render@example.com")

	nodeBody := `{"mode":"manual","name":"NAT Node","protocol":"Hysteria2","address":"nat.example.com","port":443,"listenPort":443,"publicPort":9443}`
	nodeRes := performRequest(app, http.MethodPost, "/api/v1/nodes/import", nodeBody, token)
	if nodeRes.Code != http.StatusCreated {
		t.Fatalf("expected import status 201, got %d: %s", nodeRes.Code, nodeRes.Body.String())
	}
	var node nodeResponse
	if err := json.Unmarshal(nodeRes.Body.Bytes(), &node); err != nil {
		t.Fatalf("decode node response: %v", err)
	}

	clashBody := `{"name":"Clash Subscription","format":"clash-mihomo","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	clashRes := performRequest(app, http.MethodPost, "/api/v1/subscriptions", clashBody, token)
	if clashRes.Code != http.StatusCreated {
		t.Fatalf("expected clash create status 201, got %d: %s", clashRes.Code, clashRes.Body.String())
	}
	var clash subscriptionResponse
	if err := json.Unmarshal(clashRes.Body.Bytes(), &clash); err != nil {
		t.Fatalf("decode clash subscription: %v", err)
	}
	clashPublic := performRequest(app, http.MethodGet, clash.SubscriptionURL, "", "")
	if clashPublic.Code != http.StatusOK {
		t.Fatalf("expected clash public status 200, got %d: %s", clashPublic.Code, clashPublic.Body.String())
	}
	if !strings.Contains(clashPublic.Body.String(), "mixed-port: 7890") ||
		!strings.Contains(clashPublic.Body.String(), "proxies:") ||
		!strings.Contains(clashPublic.Body.String(), "proxy-groups:") ||
		!strings.Contains(clashPublic.Body.String(), "rules:") ||
		!strings.Contains(clashPublic.Body.String(), "port: 9443") ||
		!strings.Contains(clashPublic.Body.String(), `type: "hysteria2"`) ||
		!strings.Contains(clashPublic.Body.String(), `- "NAT Node"`) {
		t.Fatalf("unexpected clash content: %s", clashPublic.Body.String())
	}

	singBody := `{"name":"Sing Subscription","format":"sing-box","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	singRes := performRequest(app, http.MethodPost, "/api/v1/subscriptions", singBody, token)
	if singRes.Code != http.StatusCreated {
		t.Fatalf("expected sing-box create status 201, got %d: %s", singRes.Code, singRes.Body.String())
	}
	var sing subscriptionResponse
	if err := json.Unmarshal(singRes.Body.Bytes(), &sing); err != nil {
		t.Fatalf("decode sing subscription: %v", err)
	}
	singPublic := performRequest(app, http.MethodGet, sing.SubscriptionURL, "", "")
	if singPublic.Code != http.StatusOK {
		t.Fatalf("expected sing public status 200, got %d: %s", singPublic.Code, singPublic.Body.String())
	}
	if !strings.Contains(singPublic.Body.String(), `"outbounds"`) || !strings.Contains(singPublic.Body.String(), `"server_port": 9443`) || !strings.Contains(singPublic.Body.String(), `"type": "hysteria2"`) {
		t.Fatalf("unexpected sing-box content: %s", singPublic.Body.String())
	}
}

func TestSubscriptionRenderingAnyTLSIncludesUUID(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-anytls", "sub-anytls@example.com")
	db := extractDB(t, nil)
	uuid := "fa6bcc36-1dbf-4f50-a811-bcd166500708"

	encryptor, err := security.NewEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
	}
	encrypted, err := encryptor.Encrypt(`{"sensitive":"{\"uuid\":\"` + uuid + `\"}"}`)
	if err != nil {
		t.Fatalf("encrypt node config: %v", err)
	}

	node := domain.ProtocolNode{
		UserID:                 1,
		Name:                   "🇯🇵 LazyCat-JP-anytls-jplite3-C06xhU",
		Protocol:               "AnyTLS",
		ListenPort:             43888,
		EncryptedProtocolJSON:  encrypted,
		SubscriptionConfigJSON: `{"address":"172.81.102.192","port":43888}`,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstallOK,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create anytls node: %v", err)
	}

	createBody := `{"name":"AnyTLS Subscription","format":"shadowrocket","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	createRes := performRequest(app, http.MethodPost, "/api/v1/subscriptions", createBody, token)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}
	var subscription subscriptionResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &subscription); err != nil {
		t.Fatalf("decode subscription response: %v", err)
	}

	publicRes := performRequest(app, http.MethodGet, subscription.SubscriptionURL, "", "")
	if publicRes.Code != http.StatusOK {
		t.Fatalf("expected public status 200, got %d: %s", publicRes.Code, publicRes.Body.String())
	}

	want := "anytls://" + uuid + "@172.81.102.192:43888?insecure=1&allowInsecure=1#%F0%9F%87%AF%F0%9F%87%B5%20LazyCat-JP-anytls-jplite3-C06xhU"
	if publicRes.Body.String() != want {
		t.Fatalf("expected anytls link %q, got %q", want, publicRes.Body.String())
	}
}

func TestSubscriptionRenderingClashAnyTLSIncludesPassword(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-clash-anytls", "sub-clash-anytls@example.com")
	db := extractDB(t, nil)
	uuid := "fa6bcc36-1dbf-4f50-a811-bcd166500708"

	encryptor, err := security.NewEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
	}
	encrypted, err := encryptor.Encrypt(`{"sensitive":"{\"uuid\":\"` + uuid + `\"}"}`)
	if err != nil {
		t.Fatalf("encrypt node config: %v", err)
	}

	node := domain.ProtocolNode{
		UserID:                 1,
		Name:                   "AnyTLS Clash Node",
		Protocol:               "AnyTLS",
		ListenPort:             43888,
		EncryptedProtocolJSON:  encrypted,
		SubscriptionConfigJSON: `{"address":"172.81.102.192","port":43888}`,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstallOK,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create anytls node: %v", err)
	}

	createBody := `{"name":"Clash AnyTLS","format":"clash-mihomo","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	createRes := performRequest(app, http.MethodPost, "/api/v1/subscriptions", createBody, token)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}
	var subscription subscriptionResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &subscription); err != nil {
		t.Fatalf("decode subscription response: %v", err)
	}

	publicRes := performRequest(app, http.MethodGet, subscription.SubscriptionURL, "", "")
	if publicRes.Code != http.StatusOK {
		t.Fatalf("expected public status 200, got %d: %s", publicRes.Code, publicRes.Body.String())
	}
	body := publicRes.Body.String()
	if !strings.Contains(body, `type: "anytls"`) ||
		!strings.Contains(body, `password: "`+uuid+`"`) ||
		!strings.Contains(body, "proxy-groups:") ||
		!strings.Contains(body, "rules:") {
		t.Fatalf("unexpected clash anytls content: %s", body)
	}
}

func TestSubscriptionRenderingClashTemplateSelection(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-clash-template", "sub-clash-template@example.com")
	node := importTestNode(t, app, token, "Template Node")

	createBody := `{"name":"Global Clash","format":"clash-mihomo","clashTemplate":"global-proxy","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	createRes := performRequest(app, http.MethodPost, "/api/v1/subscriptions", createBody, token)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}
	var subscription subscriptionResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &subscription); err != nil {
		t.Fatalf("decode subscription response: %v", err)
	}
	if subscription.ClashTemplate != "global-proxy" {
		t.Fatalf("expected global-proxy template, got %+v", subscription)
	}

	publicRes := performRequest(app, http.MethodGet, subscription.SubscriptionURL, "", "")
	if publicRes.Code != http.StatusOK {
		t.Fatalf("expected public status 200, got %d: %s", publicRes.Code, publicRes.Body.String())
	}
	body := publicRes.Body.String()
	if !strings.Contains(body, "mode: global") ||
		strings.Contains(body, "GEOIP,CN,DIRECT") ||
		!strings.Contains(body, "  - MATCH,PROXY") {
		t.Fatalf("unexpected global clash template content: %s", body)
	}
}

func TestSubscriptionRenderingCustomClashTemplate(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-custom-clash", "sub-custom-clash@example.com")
	node := importTestNode(t, app, token, "Custom Node")

	templateContent := "mixed-port: 7891\nmode: rule\nproxies:\n  - name: old\n    type: direct\nproxy-groups:\n  - name: SELECT\n    type: select\n    proxies:\n      - old\nrules:\n  - MATCH,SELECT"
	templateBody := `{"name":"Custom Template","content":` + strconv.Quote(templateContent) + `}`
	templateRes := performRequest(app, http.MethodPost, "/api/v1/clash-templates", templateBody, token)
	if templateRes.Code != http.StatusCreated {
		t.Fatalf("expected template create status 201, got %d: %s", templateRes.Code, templateRes.Body.String())
	}
	var template clashTemplateResponse
	if err := json.Unmarshal(templateRes.Body.Bytes(), &template); err != nil {
		t.Fatalf("decode template response: %v", err)
	}

	createBody := `{"name":"Custom Clash","format":"clash-mihomo","clashTemplate":"custom","clashTemplateId":` + strconvUint(template.ID) + `,"enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	createRes := performRequest(app, http.MethodPost, "/api/v1/subscriptions", createBody, token)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}
	var subscription subscriptionResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &subscription); err != nil {
		t.Fatalf("decode subscription response: %v", err)
	}
	if subscription.ClashTemplate != "custom" || subscription.ClashTemplateID == nil || *subscription.ClashTemplateID != template.ID {
		t.Fatalf("unexpected subscription template: %+v", subscription)
	}

	publicRes := performRequest(app, http.MethodGet, subscription.SubscriptionURL, "", "")
	if publicRes.Code != http.StatusOK {
		t.Fatalf("expected public status 200, got %d: %s", publicRes.Code, publicRes.Body.String())
	}
	body := publicRes.Body.String()
	if !strings.Contains(body, "mixed-port: 7891") ||
		!strings.Contains(body, `name: "Custom Node"`) ||
		strings.Contains(body, "name: old") ||
		!strings.Contains(body, `      - "Custom Node"`) ||
		!strings.Contains(body, "rules:") {
		t.Fatalf("unexpected custom clash template content: %s", body)
	}
}

func importTestNode(t *testing.T, app http.Handler, token string, name string) nodeResponse {
	t.Helper()

	body := `{"mode":"manual","name":"` + name + `","protocol":"Hysteria2","address":"example.com","port":443,"listenPort":443}`
	res := performRequest(app, http.MethodPost, "/api/v1/nodes/import", body, token)
	if res.Code != http.StatusCreated {
		t.Fatalf("import test node failed: %d %s", res.Code, res.Body.String())
	}

	var node nodeResponse
	if err := json.Unmarshal(res.Body.Bytes(), &node); err != nil {
		t.Fatalf("decode node response: %v", err)
	}
	return node
}

func strconvUint(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
