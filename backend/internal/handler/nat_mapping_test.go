package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"server-sing-box-2/backend/internal/domain"
)

type natMappingResponse struct {
	ID         uint   `json:"id"`
	ServerID   uint   `json:"serverId"`
	Name       string `json:"name"`
	Transport  string `json:"transport"`
	ListenPort int    `json:"listenPort"`
	PublicPort int    `json:"publicPort"`
	Remark     string `json:"remark"`
}

func TestNATMappingCRUD(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "nat-user", "nat-user@example.com")
	server := createTestServer(t, app, token, "NAT Server")

	createBody := `{"name":"AnyTLS Mapping","transport":"tcp","listenPort":8000,"publicPort":9000,"remark":"nat port"}`
	createRes := performRequest(app, http.MethodPost, "/api/v1/servers/1/nat-mappings", createBody, token)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}

	var created natMappingResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ServerID != server.ID || created.Transport != "TCP" || created.ListenPort != 8000 || created.PublicPort != 9000 {
		t.Fatalf("unexpected created mapping: %+v", created)
	}

	listRes := performRequest(app, http.MethodGet, "/api/v1/servers/1/nat-mappings", "", token)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d: %s", listRes.Code, listRes.Body.String())
	}

	var list []natMappingResponse
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one mapping, got %d", len(list))
	}

	updateBody := `{"name":"Updated Mapping","transport":"udp","listenPort":8001,"publicPort":9001,"remark":"updated"}`
	updateRes := performRequest(app, http.MethodPut, "/api/v1/nat-mappings/1", updateBody, token)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d: %s", updateRes.Code, updateRes.Body.String())
	}

	var updated natMappingResponse
	if err := json.Unmarshal(updateRes.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Name != "Updated Mapping" || updated.Transport != "UDP" || updated.PublicPort != 9001 {
		t.Fatalf("unexpected updated mapping: %+v", updated)
	}

	deleteRes := performRequest(app, http.MethodDelete, "/api/v1/nat-mappings/1", "", token)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("expected delete status 204, got %d", deleteRes.Code)
	}
}

func TestNATMappingRejectsCrossUserAccess(t *testing.T) {
	app := testRouter(t)
	ownerToken := registerTestUser(t, app, "nat-owner", "nat-owner@example.com")
	otherToken := registerTestUser(t, app, "nat-other", "nat-other@example.com")
	createTestServer(t, app, ownerToken, "Owner Server")

	createBody := `{"name":"Mapping","transport":"tcp","listenPort":8000,"publicPort":9000}`
	createRes := performRequest(app, http.MethodPost, "/api/v1/servers/1/nat-mappings", createBody, ownerToken)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected owner create status 201, got %d", createRes.Code)
	}

	crossList := performRequest(app, http.MethodGet, "/api/v1/servers/1/nat-mappings", "", otherToken)
	if crossList.Code != http.StatusNotFound {
		t.Fatalf("expected cross-user list status 404, got %d", crossList.Code)
	}

	crossUpdate := performRequest(app, http.MethodPut, "/api/v1/nat-mappings/1", createBody, otherToken)
	if crossUpdate.Code != http.StatusNotFound {
		t.Fatalf("expected cross-user update status 404, got %d", crossUpdate.Code)
	}

	crossDelete := performRequest(app, http.MethodDelete, "/api/v1/nat-mappings/1", "", otherToken)
	if crossDelete.Code != http.StatusNotFound {
		t.Fatalf("expected cross-user delete status 404, got %d", crossDelete.Code)
	}
}

func registerTestUser(t *testing.T, app http.Handler, username string, email string) string {
	t.Helper()

	body := `{"username":"` + username + `","email":"` + email + `","password":"password123"}`
	res := performRequest(app, http.MethodPost, "/api/v1/auth/register", body, "")
	if res.Code != http.StatusCreated {
		t.Fatalf("register test user failed: %d %s", res.Code, res.Body.String())
	}

	var response authResponse
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode auth response: %v", err)
	}
	return response.Token
}

func createTestServer(t *testing.T, app http.Handler, token string, name string) domain.Server {
	t.Helper()

	db := extractDB(t, app)
	server := domain.Server{
		UserID:      1,
		Name:        name,
		Host:        "127.0.0.1",
		SSHPort:     22,
		SSHUsername: "root",
		AuthMethod:  domain.AuthMethodPassword,
		Status:      domain.ServerStatusNormal,
	}
	if token != "" {
		var me struct {
			ID uint `json:"id"`
		}
		res := performRequest(app, http.MethodGet, "/api/v1/me", "", token)
		if res.Code != http.StatusOK {
			t.Fatalf("get me failed: %d %s", res.Code, res.Body.String())
		}
		if err := json.Unmarshal(res.Body.Bytes(), &me); err != nil {
			t.Fatalf("decode me response: %v", err)
		}
		server.UserID = me.ID
	}
	if err := db.Create(&server).Error; err != nil {
		t.Fatalf("create test server: %v", err)
	}
	return server
}
