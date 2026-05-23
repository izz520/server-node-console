package handler_test

import (
	"net/http"
	"testing"

	"server-sing-box-2/backend/internal/domain"
)

func TestDeleteServerWithoutReferences(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "server-delete", "server-delete@example.com")
	server := createTestServer(t, app, token, "Disposable Server")

	res := performRequest(app, http.MethodDelete, "/api/v1/servers/"+strconvUint(server.ID), "", token)
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected delete status 204, got %d: %s", res.Code, res.Body.String())
	}
}

func TestDeleteServerRejectsExistingNodes(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "server-node", "server-node@example.com")
	server := createTestServer(t, app, token, "Node Server")
	db := extractDB(t, app)

	node := domain.ProtocolNode{
		UserID:                 server.UserID,
		ServerID:               &server.ID,
		Name:                   "Installed Node",
		Protocol:               "Hysteria2",
		ListenPort:             443,
		SubscriptionConfigJSON: `{"address":"example.com","port":443}`,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstallOK,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	res := performRequest(app, http.MethodDelete, "/api/v1/servers/"+strconvUint(server.ID), "", token)
	if res.Code != http.StatusConflict {
		t.Fatalf("expected delete conflict for existing node, got %d: %s", res.Code, res.Body.String())
	}
}

func TestDeleteServerRejectsExistingNATMappings(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "server-nat", "server-nat@example.com")
	server := createTestServer(t, app, token, "NAT Protected Server")
	db := extractDB(t, app)

	mapping := domain.NATPortMapping{
		UserID:     server.UserID,
		ServerID:   server.ID,
		Name:       "Protected Mapping",
		Transport:  "TCP",
		ListenPort: 8000,
		PublicPort: 9000,
	}
	if err := db.Create(&mapping).Error; err != nil {
		t.Fatalf("create NAT mapping: %v", err)
	}

	res := performRequest(app, http.MethodDelete, "/api/v1/servers/"+strconvUint(server.ID), "", token)
	if res.Code != http.StatusConflict {
		t.Fatalf("expected delete conflict for NAT mapping, got %d: %s", res.Code, res.Body.String())
	}
}

func TestDeleteServerRejectsSubscriptionReferences(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "server-sub", "server-sub@example.com")
	server := createTestServer(t, app, token, "Subscription Server")
	db := extractDB(t, app)

	node := domain.ProtocolNode{
		UserID:                 server.UserID,
		ServerID:               &server.ID,
		Name:                   "Subscribed Node",
		Protocol:               "Hysteria2",
		ListenPort:             443,
		SubscriptionConfigJSON: `{"address":"example.com","port":443}`,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstallOK,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}
	subscription := domain.Subscription{
		UserID:    server.UserID,
		Name:      "Server Subscription",
		TokenHash: "server-sub-token",
		Enabled:   true,
		Format:    domain.SubscriptionFormatSingBox,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	if err := db.Create(&domain.SubscriptionNode{SubscriptionID: subscription.ID, NodeID: node.ID}).Error; err != nil {
		t.Fatalf("create subscription node: %v", err)
	}

	res := performRequest(app, http.MethodDelete, "/api/v1/servers/"+strconvUint(server.ID), "", token)
	if res.Code != http.StatusConflict {
		t.Fatalf("expected delete conflict for subscription reference, got %d: %s", res.Code, res.Body.String())
	}
}
