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

func TestSubscriptionRenderingAnyTLSIncludesExtendedParams(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-anytls-extended", "sub-anytls-extended@example.com")
	db := extractDB(t, nil)
	uuid := "1c71087d-6bee-4ce7-b619-4c8502db8b95"
	hpkp := "5E:4B:8A:96:13:C1:97:45:CF:E7:39:90:3B:06:A3:3A:AE:95:5E:EA:0B:71:6A:69:56:B8:D1:DF:DF:88:D7:09"

	encryptor, err := security.NewEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
	}
	encrypted, err := encryptor.Encrypt(`{"sensitive":"{\"uuid\":\"` + uuid + `\",\"peer\":\"addons.mozilla.org\",\"udp\":\"1\",\"hpkp\":\"` + hpkp + `\"}"}`)
	if err != nil {
		t.Fatalf("encrypt node config: %v", err)
	}

	node := domain.ProtocolNode{
		UserID:                 1,
		Name:                   "🇯🇵 Shlii-六一-JP",
		Protocol:               "AnyTLS",
		ListenPort:             21619,
		EncryptedProtocolJSON:  encrypted,
		SubscriptionConfigJSON: `{"address":"172.81.102.137","port":21619}`,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstallOK,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create anytls node: %v", err)
	}

	createBody := `{"name":"AnyTLS Extended Subscription","format":"shadowrocket","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
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

	want := "anytls://" + uuid + "@172.81.102.137:21619?peer=addons.mozilla.org&udp=1&hpkp=" + hpkp + "#%F0%9F%87%AF%F0%9F%87%B5%20Shlii-%E5%85%AD%E4%B8%80-JP"
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

func TestSubscriptionRenderingClashVLESSRealityVisionFromShareLink(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-clash-vless-reality", "sub-clash-vless-reality@example.com")

	rawLink := "vless://8660626b-d84f-4649-b206-7c065b61cf08@wj.tikle.vip:13348?type=tcp&encryption=none&security=reality&pbk=A5pF3kaVzVH2PR0Dhq2M1G28HyeHsQqus3gfHeavFyw&fp=chrome&sni=tesla.com&sid=ce4f61&spx=%2F&flow=xtls-rprx-vision#%F0%9F%87%BA%F0%9F%87%B8%20BWG-MegaBox-Pro"
	nodeBody := `{"mode":"link","rawLink":"` + rawLink + `"}`
	nodeRes := performRequest(app, http.MethodPost, "/api/v1/nodes/import", nodeBody, token)
	if nodeRes.Code != http.StatusCreated {
		t.Fatalf("expected node import status 201, got %d: %s", nodeRes.Code, nodeRes.Body.String())
	}
	var node nodeResponse
	if err := json.Unmarshal(nodeRes.Body.Bytes(), &node); err != nil {
		t.Fatalf("decode node response: %v", err)
	}

	createBody := `{"name":"Clash VLESS Reality","format":"clash-mihomo","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
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
	for _, want := range []string{
		`name: "🇺🇸 BWG-MegaBox-Pro"`,
		`type: "vless"`,
		`server: "wj.tikle.vip"`,
		`port: 13348`,
		`uuid: "8660626b-d84f-4649-b206-7c065b61cf08"`,
		`tls: true`,
		`client-fingerprint: "chrome"`,
		`servername: "tesla.com"`,
		`network: "tcp"`,
		`reality-opts:`,
		`public-key: "A5pF3kaVzVH2PR0Dhq2M1G28HyeHsQqus3gfHeavFyw"`,
		`short-id: "ce4f61"`,
		`flow: "xtls-rprx-vision"`,
		`tfo: false`,
		`skip-cert-verify: false`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected clash content to contain %q, got: %s", want, body)
		}
	}
}

func TestSubscriptionRenderingSystemVLESSRealityUsesExtractedRawLink(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-system-vless-reality", "sub-system-vless-reality@example.com")
	db := extractDB(t, nil)

	rawLink := "vless://c0464a28-9013-4e71-b21c-feb8db08dd8e@38.55.108.55:48607?encryption=none&flow=xtls-rprx-vision&security=reality&sni=apple.com&fp=chrome&pbk=Sjwj_5APjh2rKP0HC1anVN2-Ey1LtjLNq16VPn_r4Bg&sid=55cde0a4&type=tcp&headerType=none#🇺🇸 LazyCat-VMISS-Reality-vl-reality-vision-uslax-JwMH8U"
	encryptor, err := security.NewEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
	}
	encrypted, err := encryptor.Encrypt(`{"rawLink":"` + rawLink + `","sensitive":"{\"uuid\":\"c0464a28-9013-4e71-b21c-feb8db08dd8e\"}"}`)
	if err != nil {
		t.Fatalf("encrypt node config: %v", err)
	}

	node := domain.ProtocolNode{
		UserID:                 1,
		Name:                   "🇺🇸 LazyCat-VMISS-Reality",
		Protocol:               "Vless-tcp-reality-vision",
		ListenPort:             48607,
		EncryptedProtocolJSON:  encrypted,
		SubscriptionConfigJSON: `{"address":"38.55.108.55","port":48607,"generatedFrom":"argosbx"}`,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstallOK,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create vless node: %v", err)
	}

	createBody := `{"name":"System VLESS Reality","format":"shadowrocket","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
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
	if publicRes.Body.String() != rawLink {
		t.Fatalf("expected raw vless link %q, got %q", rawLink, publicRes.Body.String())
	}
}

func TestSubscriptionRenderingClashShadowsocksFromShareLink(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-clash-ss", "sub-clash-ss@example.com")

	rawLink := "ss://YWVzLTEyOC1nY206cGFzc3dvcmQ@example.com:8388?udp=true&plugin=obfs&mode=tls&host=bing.com#SS%20Node"
	nodeBody := `{"mode":"link","rawLink":"` + rawLink + `"}`
	nodeRes := performRequest(app, http.MethodPost, "/api/v1/nodes/import", nodeBody, token)
	if nodeRes.Code != http.StatusCreated {
		t.Fatalf("expected node import status 201, got %d: %s", nodeRes.Code, nodeRes.Body.String())
	}
	var node nodeResponse
	if err := json.Unmarshal(nodeRes.Body.Bytes(), &node); err != nil {
		t.Fatalf("decode node response: %v", err)
	}

	createBody := `{"name":"Clash SS","format":"clash-mihomo","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
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
	for _, want := range []string{
		`type: "ss"`,
		`cipher: "aes-128-gcm"`,
		`password: "password"`,
		`udp: true`,
		`plugin: "obfs"`,
		`plugin-opts:`,
		`mode: "tls"`,
		`host: "bing.com"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected clash content to contain %q, got: %s", want, body)
		}
	}
}

func TestSubscriptionRenderingClashInstalledShadowsocksIncludesCipherAndPassword(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-clash-installed-ss", "sub-clash-installed-ss@example.com")
	db := extractDB(t, nil)

	encryptor, err := security.NewEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
	}
	encrypted, err := encryptor.Encrypt(`{"sensitive":"{\"password\":\"lT1+yItplfIlVv3dUMyO1A==\",\"cipher\":\"2022-blake3-aes-128-gcm\"}"}`)
	if err != nil {
		t.Fatalf("encrypt node config: %v", err)
	}
	node := domain.ProtocolNode{
		UserID:                 1,
		Name:                   "Installed SS",
		Protocol:               "Shadowsocks-2022",
		ListenPort:             20886,
		EncryptedProtocolJSON:  encrypted,
		SubscriptionConfigJSON: `{"address":"185.213.17.174","port":20886,"generatedFrom":"argosbx"}`,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstallOK,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create installed shadowsocks node: %v", err)
	}

	createBody := `{"name":"Installed SS Clash","format":"clash-mihomo","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
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
	for _, want := range []string{
		`type: "ss"`,
		`server: "185.213.17.174"`,
		`port: 20886`,
		`cipher: "2022-blake3-aes-128-gcm"`,
		`password: "lT1+yItplfIlVv3dUMyO1A=="`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected clash content to contain %q, got: %s", want, body)
		}
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

func TestSubscriptionRenderingCustomClashTemplateReplacesAnchoredProxyGroups(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "sub-custom-clash-anchors", "sub-custom-clash-anchors@example.com")
	node := importTestNode(t, app, token, "Injected Node")

	templateContent := strings.Join([]string{
		"mixed-port: 7891",
		"proxies:",
		"  - name: old",
		"    type: direct",
		"proxy-groups:",
		"  - name: SELECT",
		"    type: select",
		"    proxies: &node-select",
		"      - old",
		"  - name: AUTO",
		"    type: url-test",
		"    proxies: &node-test",
		"      - old",
		"  - name: AI",
		"    type: select",
		"    proxies: *node-select",
		"rules:",
		"  - MATCH,SELECT",
	}, "\n")
	templateBody := `{"name":"Anchored Template","content":` + strconv.Quote(templateContent) + `}`
	templateRes := performRequest(app, http.MethodPost, "/api/v1/clash-templates", templateBody, token)
	if templateRes.Code != http.StatusCreated {
		t.Fatalf("expected template create status 201, got %d: %s", templateRes.Code, templateRes.Body.String())
	}
	var template clashTemplateResponse
	if err := json.Unmarshal(templateRes.Body.Bytes(), &template); err != nil {
		t.Fatalf("decode template response: %v", err)
	}

	createBody := `{"name":"Anchored Clash","format":"clash-mihomo","clashTemplate":"custom","clashTemplateId":` + strconvUint(template.ID) + `,"enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
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
	if !strings.Contains(body, "    proxies: &node-select\n      - \"Injected Node\"\n      - DIRECT") ||
		!strings.Contains(body, "    proxies: &node-test\n      - \"Injected Node\"\n      - DIRECT") ||
		!strings.Contains(body, "    proxies: *node-select") ||
		strings.Contains(body, "      - old") {
		t.Fatalf("unexpected anchored custom clash template content: %s", body)
	}
}

func TestClashSubscriptionIncludesChainProxyDependency(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "chain-proxy", "chain-proxy@example.com")
	upstream := importTestNode(t, app, token, "Upstream Node")
	downstream := importTestNode(t, app, token, "Downstream Node")

	updateBody := `{"name":"Downstream Node","protocol":"Hysteria2","address":"example.com","port":443,"listenPort":443,"chainProxyNodeId":` + strconvUint(upstream.ID) + `}`
	updateRes := performRequest(app, http.MethodPut, "/api/v1/nodes/"+strconvUint(downstream.ID), updateBody, token)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("update downstream chain proxy failed: %d %s", updateRes.Code, updateRes.Body.String())
	}

	subscriptionBody := `{"name":"Chain Clash","format":"clash-mihomo","clashTemplate":"rule-cn","enabled":true,"nodeIds":[` + strconvUint(downstream.ID) + `]}`
	res := performRequest(app, http.MethodPost, "/api/v1/subscriptions", subscriptionBody, token)
	if res.Code != http.StatusCreated {
		t.Fatalf("create subscription failed: %d %s", res.Code, res.Body.String())
	}
	var subscription subscriptionResponse
	if err := json.Unmarshal(res.Body.Bytes(), &subscription); err != nil {
		t.Fatalf("decode subscription: %v", err)
	}

	publicRes := performRequest(app, http.MethodGet, "/sub/"+subscription.Token, "", "")
	body := publicRes.Body.String()
	if !strings.Contains(body, `name: "Upstream Node"`) ||
		!strings.Contains(body, `name: "Downstream Node"`) ||
		!strings.Contains(body, `dialer-proxy: "Upstream Node"`) {
		t.Fatalf("expected chain proxy dependency and dialer-proxy in clash output: %s", body)
	}
}

func TestClashSubscriptionParsesSocks5CredentialsFromRawLink(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "socks5-credentials", "socks5-credentials@example.com")
	rawLink := "socks5://20ri3UoV:Q1cLg3fj@163.123.202.206:5491#%F0%9F%87%BA%F0%9F%87%B8%20Webshare-US-ISP"
	nodeBody := `{"mode":"link","rawLink":"` + rawLink + `"}`
	nodeRes := performRequest(app, http.MethodPost, "/api/v1/nodes/import", nodeBody, token)
	if nodeRes.Code != http.StatusCreated {
		t.Fatalf("import socks5 node failed: %d %s", nodeRes.Code, nodeRes.Body.String())
	}
	var node nodeResponse
	if err := json.Unmarshal(nodeRes.Body.Bytes(), &node); err != nil {
		t.Fatalf("decode node: %v", err)
	}

	subscriptionBody := `{"name":"Socks Clash","format":"clash-mihomo","clashTemplate":"rule-cn","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	res := performRequest(app, http.MethodPost, "/api/v1/subscriptions", subscriptionBody, token)
	if res.Code != http.StatusCreated {
		t.Fatalf("create subscription failed: %d %s", res.Code, res.Body.String())
	}
	var subscription subscriptionResponse
	if err := json.Unmarshal(res.Body.Bytes(), &subscription); err != nil {
		t.Fatalf("decode subscription: %v", err)
	}

	publicRes := performRequest(app, http.MethodGet, "/sub/"+subscription.Token, "", "")
	body := publicRes.Body.String()
	if !strings.Contains(body, `username: "20ri3UoV"`) ||
		!strings.Contains(body, `password: "Q1cLg3fj"`) {
		t.Fatalf("expected socks5 credentials in clash output: %s", body)
	}
}

func TestClashSubscriptionRendersAnyTLSHPKPAsFingerprint(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "anytls-hpkp", "anytls-hpkp@example.com")
	rawLink := "anytls://7626bc8a-22c6-4c69-af8d-10357cb9145b@43.255.159.75:20844?peer=addons.mozilla.org&udp=1&hpkp=AA:9F:FC:25:A2:EF:D2:9E:F0:D4:A8:39:A2:C2:7C:C0:B8:CF:39:C4:6A:71:70:FD:E7:C1:3A:10:97:AB:32:23#%F0%9F%87%AD%F0%9F%87%B0%205SSR-HK-FREE"
	nodeBody := `{"mode":"link","rawLink":"` + rawLink + `"}`
	nodeRes := performRequest(app, http.MethodPost, "/api/v1/nodes/import", nodeBody, token)
	if nodeRes.Code != http.StatusCreated {
		t.Fatalf("import anytls node failed: %d %s", nodeRes.Code, nodeRes.Body.String())
	}
	var node nodeResponse
	if err := json.Unmarshal(nodeRes.Body.Bytes(), &node); err != nil {
		t.Fatalf("decode node: %v", err)
	}

	subscriptionBody := `{"name":"AnyTLS Clash","format":"clash-mihomo","clashTemplate":"rule-cn","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	res := performRequest(app, http.MethodPost, "/api/v1/subscriptions", subscriptionBody, token)
	if res.Code != http.StatusCreated {
		t.Fatalf("create subscription failed: %d %s", res.Code, res.Body.String())
	}
	var subscription subscriptionResponse
	if err := json.Unmarshal(res.Body.Bytes(), &subscription); err != nil {
		t.Fatalf("decode subscription: %v", err)
	}

	publicRes := performRequest(app, http.MethodGet, "/sub/"+subscription.Token, "", "")
	body := publicRes.Body.String()
	for _, want := range []string{
		`password: "7626bc8a-22c6-4c69-af8d-10357cb9145b"`,
		`client-fingerprint: "firefox"`,
		`udp: true`,
		`idle-session-check-interval: 30`,
		`idle-session-timeout: 30`,
		`sni: "addons.mozilla.org"`,
		`fingerprint: "AA:9F:FC:25:A2:EF:D2:9E:F0:D4:A8:39:A2:C2:7C:C0:B8:CF:39:C4:6A:71:70:FD:E7:C1:3A:10:97:AB:32:23"`,
		`skip-cert-verify: false`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected anytls clash output to contain %q, got: %s", want, body)
		}
	}
}

func TestClashSubscriptionPreservesUnknownSafeQueryFields(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "unknown-query-fields", "unknown-query-fields@example.com")
	rawLink := "socks5://user:pass@127.0.0.1:1080?udp=1&dialer-proxy=Upstream&custom-field=kept&unsafe.field=drop#Local-Socks"
	nodeBody := `{"mode":"link","rawLink":"` + rawLink + `"}`
	nodeRes := performRequest(app, http.MethodPost, "/api/v1/nodes/import", nodeBody, token)
	if nodeRes.Code != http.StatusCreated {
		t.Fatalf("import socks5 node failed: %d %s", nodeRes.Code, nodeRes.Body.String())
	}
	var node nodeResponse
	if err := json.Unmarshal(nodeRes.Body.Bytes(), &node); err != nil {
		t.Fatalf("decode node: %v", err)
	}

	subscriptionBody := `{"name":"Unknown Fields","format":"clash-mihomo","clashTemplate":"rule-cn","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	res := performRequest(app, http.MethodPost, "/api/v1/subscriptions", subscriptionBody, token)
	if res.Code != http.StatusCreated {
		t.Fatalf("create subscription failed: %d %s", res.Code, res.Body.String())
	}
	var subscription subscriptionResponse
	if err := json.Unmarshal(res.Body.Bytes(), &subscription); err != nil {
		t.Fatalf("decode subscription: %v", err)
	}

	publicRes := performRequest(app, http.MethodGet, "/sub/"+subscription.Token, "", "")
	body := publicRes.Body.String()
	if !strings.Contains(body, `custom-field: "kept"`) ||
		!strings.Contains(body, `dialer-proxy: "Upstream"`) ||
		strings.Contains(body, `unsafe.field`) {
		t.Fatalf("expected safe unknown query fields to be preserved: %s", body)
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
