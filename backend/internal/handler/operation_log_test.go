package handler_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"server-sing-box-2/backend/internal/domain"
)

type operationLogTestResponse struct {
	ID        uint       `json:"id"`
	UserID    *uint      `json:"userId"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	Metadata  string     `json:"metadata"`
	CreatedAt *time.Time `json:"createdAt"`
}

func TestOperationLogsAreScopedToCurrentUser(t *testing.T) {
	app := testRouter(t)
	registerTestUser(t, app, "log-owner", "log-owner@example.com")
	otherToken := registerTestUser(t, app, "log-other", "log-other@example.com")

	loginRes := performRequest(app, http.MethodPost, "/api/v1/auth/login", `{"account":"log-owner","password":"password123"}`, "")
	if loginRes.Code != http.StatusOK {
		t.Fatalf("owner login failed: %d %s", loginRes.Code, loginRes.Body.String())
	}
	var auth authResponse
	if err := json.Unmarshal(loginRes.Body.Bytes(), &auth); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	ownerRes := performRequest(app, http.MethodGet, "/api/v1/operation-logs", "", auth.Token)
	if ownerRes.Code != http.StatusOK {
		t.Fatalf("expected owner operation logs status 200, got %d: %s", ownerRes.Code, ownerRes.Body.String())
	}
	var ownerLogs []operationLogTestResponse
	if err := json.Unmarshal(ownerRes.Body.Bytes(), &ownerLogs); err != nil {
		t.Fatalf("decode owner logs: %v", err)
	}
	if len(ownerLogs) != 1 {
		t.Fatalf("expected one owner log, got %d", len(ownerLogs))
	}
	if ownerLogs[0].Action != "auth.login" || ownerLogs[0].Resource != "user" {
		t.Fatalf("unexpected owner log: %+v", ownerLogs[0])
	}
	if strings.Contains(ownerLogs[0].Metadata, "password123") {
		t.Fatal("operation log metadata must not include sensitive password")
	}

	otherRes := performRequest(app, http.MethodGet, "/api/v1/operation-logs", "", otherToken)
	if otherRes.Code != http.StatusOK {
		t.Fatalf("expected other operation logs status 200, got %d: %s", otherRes.Code, otherRes.Body.String())
	}
	var otherLogs []operationLogTestResponse
	if err := json.Unmarshal(otherRes.Body.Bytes(), &otherLogs); err != nil {
		t.Fatalf("decode other logs: %v", err)
	}
	if len(otherLogs) != 0 {
		t.Fatalf("expected no logs for other user, got %d", len(otherLogs))
	}
}

func TestAdminCanReadAllOperationLogs(t *testing.T) {
	app := testRouter(t)
	registerTestUser(t, app, "log-admin", "log-admin@example.com")
	userToken := registerTestUser(t, app, "log-normal", "log-normal@example.com")

	db := extractDB(t, app)
	if err := db.Model(&domain.User{}).Where("username = ?", "log-admin").Update("role", domain.UserRoleAdmin).Error; err != nil {
		t.Fatalf("promote admin: %v", err)
	}

	userLoginRes := performRequest(app, http.MethodPost, "/api/v1/auth/login", `{"account":"log-normal","password":"password123"}`, "")
	if userLoginRes.Code != http.StatusOK {
		t.Fatalf("normal login failed: %d %s", userLoginRes.Code, userLoginRes.Body.String())
	}
	adminLoginRes := performRequest(app, http.MethodPost, "/api/v1/auth/login", `{"account":"log-admin","password":"password123"}`, "")
	if adminLoginRes.Code != http.StatusOK {
		t.Fatalf("admin login failed: %d %s", adminLoginRes.Code, adminLoginRes.Body.String())
	}
	var adminAuth authResponse
	if err := json.Unmarshal(adminLoginRes.Body.Bytes(), &adminAuth); err != nil {
		t.Fatalf("decode admin login: %v", err)
	}

	forbiddenRes := performRequest(app, http.MethodGet, "/api/v1/admin/operation-logs", "", userToken)
	if forbiddenRes.Code != http.StatusForbidden {
		t.Fatalf("expected normal user admin logs status 403, got %d", forbiddenRes.Code)
	}

	adminRes := performRequest(app, http.MethodGet, "/api/v1/admin/operation-logs", "", adminAuth.Token)
	if adminRes.Code != http.StatusOK {
		t.Fatalf("expected admin operation logs status 200, got %d: %s", adminRes.Code, adminRes.Body.String())
	}
	var logs []operationLogTestResponse
	if err := json.Unmarshal(adminRes.Body.Bytes(), &logs); err != nil {
		t.Fatalf("decode admin logs: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected admin to see all user logs, got %d", len(logs))
	}
}
