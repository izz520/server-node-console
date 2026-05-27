package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"server-sing-box-2/backend/internal/domain"
)

func TestInstallNodeCreatesNodeAndTask(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "install-user", "install-user@example.com")
	server := createTestServer(t, app, token, "Install Server")

	body := `{"serverId":` + strconvUint(server.ID) + `,"name":"AnyTLS Node","protocol":"AnyTLS","port":8443}`
	res := performRequest(app, http.MethodPost, "/api/v1/nodes/install", body, token)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected install status 202, got %d: %s", res.Code, res.Body.String())
	}

	var response struct {
		Node nodeResponse     `json:"node"`
		Task taskListResponse `json:"task"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode install response: %v", err)
	}
	if response.Node.Status != string(domain.NodeStatusInstalling) || response.Task.Status != string(domain.TaskStatusQueued) {
		t.Fatalf("unexpected install response: %+v", response)
	}
}

func TestInstallNodeGeneratesDefaultPortAndSensitiveParams(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "install-default", "install-default@example.com")
	server := createTestServer(t, app, token, "Default Install Server")

	body := `{"serverId":` + strconvUint(server.ID) + `,"name":"Vless Node","protocol":"Vless-tcp-reality-vision"}`
	res := performRequest(app, http.MethodPost, "/api/v1/nodes/install", body, token)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected install status 202, got %d: %s", res.Code, res.Body.String())
	}

	var response struct {
		Node nodeResponse     `json:"node"`
		Task taskListResponse `json:"task"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode install response: %v", err)
	}
	if response.Node.Port < 20000 || response.Node.Port > 49999 {
		t.Fatalf("expected generated high port, got %d", response.Node.Port)
	}
	if response.Node.ListenPort != response.Node.Port {
		t.Fatalf("expected listen port to match generated port, got node=%+v", response.Node)
	}
	if !response.Node.HasSensitive {
		t.Fatal("expected generated UUID to be encrypted and marked as sensitive")
	}
}

func TestInstallShadowsocksGeneratesSensitiveParams(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "install-ss-default", "install-ss-default@example.com")
	server := createTestServer(t, app, token, "SS Default Install Server")

	body := `{"serverId":` + strconvUint(server.ID) + `,"name":"SS Node","protocol":"Shadowsocks-2022","port":20886}`
	res := performRequest(app, http.MethodPost, "/api/v1/nodes/install", body, token)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected install status 202, got %d: %s", res.Code, res.Body.String())
	}

	var response struct {
		Node nodeResponse     `json:"node"`
		Task taskListResponse `json:"task"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode install response: %v", err)
	}
	if !response.Node.HasSensitive {
		t.Fatal("expected shadowsocks password and cipher to be encrypted and marked as sensitive")
	}
}

func TestInstallNodeStoresPublicSubscriptionPort(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "install-public-port", "install-public-port@example.com")
	server := createTestServer(t, app, token, "Public Port Install Server")

	body := `{"serverId":` + strconvUint(server.ID) + `,"name":"AnyTLS Node","protocol":"AnyTLS","port":43888,"publicPort":48888}`
	res := performRequest(app, http.MethodPost, "/api/v1/nodes/install", body, token)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected install status 202, got %d: %s", res.Code, res.Body.String())
	}

	var response struct {
		Node nodeResponse     `json:"node"`
		Task taskListResponse `json:"task"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode install response: %v", err)
	}
	if response.Node.PublicPort == nil || *response.Node.PublicPort != 48888 {
		t.Fatalf("expected public subscription port 48888, got %+v", response.Node)
	}
}

func TestInstallFixedArgoRequiresDomainAndToken(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "install-argo-required", "install-argo-required@example.com")
	server := createTestServer(t, app, token, "Argo Required Server")

	body := `{"serverId":` + strconvUint(server.ID) + `,"name":"Argo Node","protocol":"Argo 固定隧道","port":20888}`
	res := performRequest(app, http.MethodPost, "/api/v1/nodes/install", body, token)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected fixed argo validation status 400, got %d: %s", res.Code, res.Body.String())
	}
}

func TestInstallFixedArgoAcceptsDisplayProtocolName(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "install-argo-display", "install-argo-display@example.com")
	server := createTestServer(t, app, token, "Argo Display Server")

	body := `{"serverId":` + strconvUint(server.ID) + `,"name":"Argo Node","protocol":"Argo 固定隧道","port":20888,"argoDomain":"tunnel.example.com","argoToken":"token-value"}`
	res := performRequest(app, http.MethodPost, "/api/v1/nodes/install", body, token)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected fixed argo install status 202, got %d: %s", res.Code, res.Body.String())
	}
}

func TestInstallNodeRejectsOtherUserServer(t *testing.T) {
	app := testRouter(t)
	ownerToken := registerTestUser(t, app, "install-owner", "install-owner@example.com")
	otherToken := registerTestUser(t, app, "install-other", "install-other@example.com")
	server := createTestServer(t, app, ownerToken, "Owner Server")

	body := `{"serverId":` + strconvUint(server.ID) + `,"name":"AnyTLS Node","protocol":"AnyTLS","port":8443}`
	res := performRequest(app, http.MethodPost, "/api/v1/nodes/install", body, otherToken)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected cross-user install status 404, got %d", res.Code)
	}
}

func TestInstallNodeRejectsNonNormalServer(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "install-status", "install-status@example.com")
	server := createTestServer(t, app, token, "Failed Server")

	db := extractDB(t, app)
	if err := db.Model(&domain.Server{}).Where("id = ?", server.ID).Update("status", domain.ServerStatusConnectionFailed).Error; err != nil {
		t.Fatalf("update server status: %v", err)
	}

	body := `{"serverId":` + strconvUint(server.ID) + `,"name":"AnyTLS Node","protocol":"AnyTLS","port":8443}`
	res := performRequest(app, http.MethodPost, "/api/v1/nodes/install", body, token)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected non-normal server install status 400, got %d: %s", res.Code, res.Body.String())
	}
}

func TestUninstallNodeCreatesTask(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "uninstall-user", "uninstall-user@example.com")
	server := createTestServer(t, app, token, "Uninstall Server")
	db := extractDB(t, nil)

	node := domain.ProtocolNode{
		UserID:                 server.UserID,
		ServerID:               &server.ID,
		Name:                   "Installed Node",
		Protocol:               "AnyTLS",
		ListenPort:             8443,
		SubscriptionConfigJSON: `{"address":"127.0.0.1","port":8443}`,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstallOK,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create installed node: %v", err)
	}

	res := performRequest(app, http.MethodPost, "/api/v1/nodes/1/uninstall", "", token)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected uninstall status 202, got %d: %s", res.Code, res.Body.String())
	}

	var response struct {
		Node nodeResponse     `json:"node"`
		Task taskListResponse `json:"task"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode uninstall response: %v", err)
	}
	if response.Node.Status != string(domain.NodeStatusUninstalling) || response.Task.Type != string(domain.TaskTypeUninstall) {
		t.Fatalf("unexpected uninstall response: %+v", response)
	}
}

func TestUninstallNodeAcceptsDeleteAfterUninstall(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "uninstall-delete-user", "uninstall-delete-user@example.com")
	server := createTestServer(t, app, token, "Uninstall Delete Server")
	db := extractDB(t, nil)

	node := domain.ProtocolNode{
		UserID:                 server.UserID,
		ServerID:               &server.ID,
		Name:                   "Installed Node",
		Protocol:               "AnyTLS",
		ListenPort:             8443,
		SubscriptionConfigJSON: `{"address":"127.0.0.1","port":8443}`,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstallOK,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create installed node: %v", err)
	}

	res := performRequest(app, http.MethodPost, "/api/v1/nodes/"+strconvUint(node.ID)+"/uninstall", `{"deleteAfterUninstall":true}`, token)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected uninstall status 202, got %d: %s", res.Code, res.Body.String())
	}

	var response struct {
		Node nodeResponse     `json:"node"`
		Task taskListResponse `json:"task"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode uninstall response: %v", err)
	}
	if response.Node.Status != string(domain.NodeStatusUninstalling) || response.Task.Type != string(domain.TaskTypeUninstall) {
		t.Fatalf("unexpected uninstall response: %+v", response)
	}
}

func TestUninstallNodeRejectsNonNormalServer(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "uninstall-status", "uninstall-status@example.com")
	server := createTestServer(t, app, token, "Failed Uninstall Server")
	db := extractDB(t, nil)

	node := domain.ProtocolNode{
		UserID:                 server.UserID,
		ServerID:               &server.ID,
		Name:                   "Installed Node",
		Protocol:               "AnyTLS",
		ListenPort:             8443,
		SubscriptionConfigJSON: `{"address":"127.0.0.1","port":8443}`,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstallOK,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create installed node: %v", err)
	}
	if err := db.Model(&domain.Server{}).Where("id = ?", server.ID).Update("status", domain.ServerStatusConnectionFailed).Error; err != nil {
		t.Fatalf("update server status: %v", err)
	}

	res := performRequest(app, http.MethodPost, "/api/v1/nodes/"+strconvUint(node.ID)+"/uninstall", "", token)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected non-normal server uninstall status 400, got %d: %s", res.Code, res.Body.String())
	}
}
