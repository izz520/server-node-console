package handler_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"server-sing-box-2/backend/internal/domain"
)

type nodeResponse struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	Protocol      string `json:"protocol"`
	Address       string `json:"address"`
	Port          int    `json:"port"`
	ListenPort    int    `json:"listenPort"`
	InstallMethod string `json:"installMethod"`
	Status        string `json:"status"`
	HasSensitive  bool   `json:"hasSensitive"`
}

func TestExternalNodeCRUD(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "node-user", "node-user@example.com")

	createBody := `{"mode":"manual","name":"External Hysteria2","protocol":"Hysteria2","address":"example.com","port":443,"listenPort":443,"sensitive":"password=secret"}`
	createRes := performRequest(app, http.MethodPost, "/api/v1/nodes/import", createBody, token)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}

	var created nodeResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Protocol != "Hysteria2" || created.Address != "example.com" || !created.HasSensitive {
		t.Fatalf("unexpected created node: %+v", created)
	}

	listRes := performRequest(app, http.MethodGet, "/api/v1/nodes", "", token)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d: %s", listRes.Code, listRes.Body.String())
	}

	var list []nodeResponse
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one node, got %d", len(list))
	}

	updateBody := `{"name":"Updated Node","protocol":"Tuic","address":"node.example.com","port":8443,"listenPort":8443,"remark":"updated"}`
	updateRes := performRequest(app, http.MethodPut, "/api/v1/nodes/1", updateBody, token)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d: %s", updateRes.Code, updateRes.Body.String())
	}

	var updated nodeResponse
	if err := json.Unmarshal(updateRes.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Name != "Updated Node" || updated.Protocol != "Tuic" || updated.Port != 8443 {
		t.Fatalf("unexpected updated node: %+v", updated)
	}

	deleteRes := performRequest(app, http.MethodDelete, "/api/v1/nodes/1", "", token)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("expected delete status 204, got %d", deleteRes.Code)
	}
}

func TestImportNodeFromShareLink(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "node-link-user", "node-link-user@example.com")

	body := `{"mode":"link","rawLink":"vless://uuid@example.com:443?security=reality#Reality%20Node"}`
	res := performRequest(app, http.MethodPost, "/api/v1/nodes/import", body, token)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected link import status 201, got %d: %s", res.Code, res.Body.String())
	}

	var node nodeResponse
	if err := json.Unmarshal(res.Body.Bytes(), &node); err != nil {
		t.Fatalf("decode link node response: %v", err)
	}
	if node.Protocol != "Vless" || node.Address != "example.com" || node.Port != 443 || !node.HasSensitive {
		t.Fatalf("unexpected link node: %+v", node)
	}
}

func TestImportVMessNodeFromShareLink(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "node-vmess-user", "node-vmess-user@example.com")

	payload := `{"ps":"VMess Node","add":"vmess.example.com","port":"443"}`
	link := "vmess://" + base64.StdEncoding.EncodeToString([]byte(payload))
	body := `{"mode":"link","rawLink":"` + link + `"}`
	res := performRequest(app, http.MethodPost, "/api/v1/nodes/import", body, token)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected vmess import status 201, got %d: %s", res.Code, res.Body.String())
	}

	var node nodeResponse
	if err := json.Unmarshal(res.Body.Bytes(), &node); err != nil {
		t.Fatalf("decode vmess node response: %v", err)
	}
	if node.Protocol != "Vmess-ws" || node.Address != "vmess.example.com" || node.Port != 443 {
		t.Fatalf("unexpected vmess node: %+v", node)
	}
}

func TestDeleteNodeRemovesSubscriptionLinks(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "node-sub-link", "node-sub-link@example.com")

	createBody := `{"mode":"manual","name":"Subscribed Node","protocol":"Hysteria2","address":"example.com","port":443}`
	createRes := performRequest(app, http.MethodPost, "/api/v1/nodes/import", createBody, token)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected node create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}
	var node nodeResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &node); err != nil {
		t.Fatalf("decode node response: %v", err)
	}

	subscriptionBody := `{"name":"Node Link Subscription","format":"sing-box","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	subscriptionRes := performRequest(app, http.MethodPost, "/api/v1/subscriptions", subscriptionBody, token)
	if subscriptionRes.Code != http.StatusCreated {
		t.Fatalf("expected subscription create status 201, got %d: %s", subscriptionRes.Code, subscriptionRes.Body.String())
	}

	deleteRes := performRequest(app, http.MethodDelete, "/api/v1/nodes/"+strconvUint(node.ID), "", token)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("expected delete status 204, got %d: %s", deleteRes.Code, deleteRes.Body.String())
	}

	getSubscriptionRes := performRequest(app, http.MethodGet, "/api/v1/subscriptions/1", "", token)
	if getSubscriptionRes.Code != http.StatusOK {
		t.Fatalf("expected get subscription status 200, got %d: %s", getSubscriptionRes.Code, getSubscriptionRes.Body.String())
	}
	var subscription subscriptionResponse
	if err := json.Unmarshal(getSubscriptionRes.Body.Bytes(), &subscription); err != nil {
		t.Fatalf("decode subscription response: %v", err)
	}
	if subscription.NodeCount != 0 || len(subscription.NodeIDs) != 0 {
		t.Fatalf("expected subscription links to be removed, got %+v", subscription)
	}
}

func TestDeleteNodeRejectsInstalledSystemNode(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "node-system-delete", "node-system-delete@example.com")
	server := createTestServer(t, app, token, "System Node Server")
	db := extractDB(t, app)

	node := domain.ProtocolNode{
		UserID:                 server.UserID,
		ServerID:               &server.ID,
		Name:                   "Installed System Node",
		Protocol:               "AnyTLS",
		ListenPort:             8443,
		SubscriptionConfigJSON: `{"address":"127.0.0.1","port":8443}`,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstallOK,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create system node: %v", err)
	}

	deleteRes := performRequest(app, http.MethodDelete, "/api/v1/nodes/"+strconvUint(node.ID), "", token)
	if deleteRes.Code != http.StatusBadRequest {
		t.Fatalf("expected delete installed system node status 400, got %d: %s", deleteRes.Code, deleteRes.Body.String())
	}
}

func TestNodeRejectsCrossUserAccess(t *testing.T) {
	app := testRouter(t)
	ownerToken := registerTestUser(t, app, "node-owner", "node-owner@example.com")
	otherToken := registerTestUser(t, app, "node-other", "node-other@example.com")

	createBody := `{"mode":"manual","name":"Owner Node","protocol":"Socks5","address":"example.com","port":1080}`
	createRes := performRequest(app, http.MethodPost, "/api/v1/nodes/import", createBody, ownerToken)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected owner create status 201, got %d", createRes.Code)
	}

	listRes := performRequest(app, http.MethodGet, "/api/v1/nodes", "", otherToken)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected other list status 200, got %d", listRes.Code)
	}
	var list []nodeResponse
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode other list response: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected other user to see zero nodes, got %d", len(list))
	}

	updateBody := `{"name":"Hack","protocol":"Socks5","address":"other.example.com","port":1080}`
	updateRes := performRequest(app, http.MethodPut, "/api/v1/nodes/1", updateBody, otherToken)
	if updateRes.Code != http.StatusNotFound {
		t.Fatalf("expected cross-user update status 404, got %d", updateRes.Code)
	}

	deleteRes := performRequest(app, http.MethodDelete, "/api/v1/nodes/1", "", otherToken)
	if deleteRes.Code != http.StatusNotFound {
		t.Fatalf("expected cross-user delete status 404, got %d", deleteRes.Code)
	}
}
