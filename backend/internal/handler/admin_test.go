package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"server-sing-box-2/backend/internal/domain"
)

func TestAdminReadOnlyEndpoints(t *testing.T) {
	app := testRouter(t)
	adminToken := registerTestUser(t, app, "admin", "admin@example.com")
	userToken := registerTestUser(t, app, "normal", "normal@example.com")

	db := extractDB(t, nil)
	if err := db.Model(&domain.User{}).Where("username = ?", "admin").Update("role", domain.UserRoleAdmin).Error; err != nil {
		t.Fatalf("promote admin: %v", err)
	}

	// Login again so the JWT carries the updated role.
	loginRes := performRequest(app, http.MethodPost, "/api/v1/auth/login", `{"account":"admin","password":"password123"}`, "")
	if loginRes.Code != http.StatusOK {
		t.Fatalf("admin login failed: %d %s", loginRes.Code, loginRes.Body.String())
	}
	var auth authResponse
	if err := json.Unmarshal(loginRes.Body.Bytes(), &auth); err != nil {
		t.Fatalf("decode admin login: %v", err)
	}
	adminToken = auth.Token

	for _, path := range []string{
		"/api/v1/admin/users",
		"/api/v1/admin/servers",
		"/api/v1/admin/nodes",
		"/api/v1/admin/subscriptions",
		"/api/v1/admin/tasks",
		"/api/v1/admin/operation-logs",
	} {
		adminRes := performRequest(app, http.MethodGet, path, "", adminToken)
		if adminRes.Code != http.StatusOK {
			t.Fatalf("expected admin access 200 for %s, got %d: %s", path, adminRes.Code, adminRes.Body.String())
		}

		userRes := performRequest(app, http.MethodGet, path, "", userToken)
		if userRes.Code != http.StatusForbidden {
			t.Fatalf("expected user access 403 for %s, got %d", path, userRes.Code)
		}
	}
}

func TestAdminSubscriptionsDoNotExposeToken(t *testing.T) {
	app := testRouter(t)
	registerTestUser(t, app, "admin-sub", "admin-sub@example.com")
	userToken := registerTestUser(t, app, "owner-sub", "owner-sub@example.com")
	node := importTestNode(t, app, userToken, "Admin Hidden Token Node")

	createBody := `{"name":"Hidden Token Subscription","format":"sing-box","enabled":true,"nodeIds":[` + strconvUint(node.ID) + `]}`
	createRes := performRequest(app, http.MethodPost, "/api/v1/subscriptions", createBody, userToken)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected subscription create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}

	db := extractDB(t, nil)
	if err := db.Model(&domain.User{}).Where("username = ?", "admin-sub").Update("role", domain.UserRoleAdmin).Error; err != nil {
		t.Fatalf("promote admin: %v", err)
	}
	loginRes := performRequest(app, http.MethodPost, "/api/v1/auth/login", `{"account":"admin-sub","password":"password123"}`, "")
	if loginRes.Code != http.StatusOK {
		t.Fatalf("admin login failed: %d %s", loginRes.Code, loginRes.Body.String())
	}
	var auth authResponse
	if err := json.Unmarshal(loginRes.Body.Bytes(), &auth); err != nil {
		t.Fatalf("decode admin login: %v", err)
	}

	adminRes := performRequest(app, http.MethodGet, "/api/v1/admin/subscriptions", "", auth.Token)
	if adminRes.Code != http.StatusOK {
		t.Fatalf("expected admin subscriptions status 200, got %d: %s", adminRes.Code, adminRes.Body.String())
	}
	var subscriptions []subscriptionResponse
	if err := json.Unmarshal(adminRes.Body.Bytes(), &subscriptions); err != nil {
		t.Fatalf("decode admin subscriptions: %v", err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("expected one subscription, got %d", len(subscriptions))
	}
	if subscriptions[0].Token != "" || subscriptions[0].SubscriptionURL != "" {
		t.Fatalf("admin subscription response exposed token: %+v", subscriptions[0])
	}
}
