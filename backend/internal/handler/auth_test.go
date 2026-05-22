package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"server-sing-box-2/backend/internal/config"
	"server-sing-box-2/backend/internal/domain"
	"server-sing-box-2/backend/internal/router"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type authResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	} `json:"user"`
}

func TestAuthRegisterLoginMeAndRefresh(t *testing.T) {
	app := testRouter(t)

	registerBody := `{"username":"alice","email":"alice@example.com","password":"password123"}`
	registerRes := performRequest(app, http.MethodPost, "/api/v1/auth/register", registerBody, "")
	if registerRes.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d: %s", registerRes.Code, registerRes.Body.String())
	}

	var registered authResponse
	if err := json.Unmarshal(registerRes.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if registered.Token == "" {
		t.Fatal("expected register token")
	}
	if registered.User.Role != string(domain.UserRoleUser) {
		t.Fatalf("expected default user role, got %q", registered.User.Role)
	}
	if ttl := time.Until(registered.ExpiresAt); ttl < 6*24*time.Hour || ttl > 7*24*time.Hour+time.Minute {
		t.Fatalf("expected token ttl close to 7 days, got %s", ttl)
	}

	duplicateRes := performRequest(app, http.MethodPost, "/api/v1/auth/register", registerBody, "")
	if duplicateRes.Code != http.StatusConflict {
		t.Fatalf("expected duplicate status 409, got %d", duplicateRes.Code)
	}

	loginBody := `{"account":"alice@example.com","password":"password123"}`
	loginRes := performRequest(app, http.MethodPost, "/api/v1/auth/login", loginBody, "")
	if loginRes.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d: %s", loginRes.Code, loginRes.Body.String())
	}

	var loggedIn authResponse
	if err := json.Unmarshal(loginRes.Body.Bytes(), &loggedIn); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loggedIn.Token == "" {
		t.Fatal("expected login token")
	}

	meRes := performRequest(app, http.MethodGet, "/api/v1/me", "", loggedIn.Token)
	if meRes.Code != http.StatusOK {
		t.Fatalf("expected me status 200, got %d: %s", meRes.Code, meRes.Body.String())
	}

	refreshRes := performRequest(app, http.MethodPost, "/api/v1/auth/refresh", "", loggedIn.Token)
	if refreshRes.Code != http.StatusOK {
		t.Fatalf("expected refresh status 200, got %d: %s", refreshRes.Code, refreshRes.Body.String())
	}

	var refreshed authResponse
	if err := json.Unmarshal(refreshRes.Body.Bytes(), &refreshed); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if refreshed.Token == "" {
		t.Fatal("expected refreshed token")
	}
}

func TestAuthRejectsInvalidCredentialsAndToken(t *testing.T) {
	app := testRouter(t)

	registerBody := `{"username":"bob","email":"bob@example.com","password":"password123"}`
	if res := performRequest(app, http.MethodPost, "/api/v1/auth/register", registerBody, ""); res.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", res.Code)
	}

	loginBody := `{"account":"bob@example.com","password":"wrong-password"}`
	if res := performRequest(app, http.MethodPost, "/api/v1/auth/login", loginBody, ""); res.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid login status 401, got %d", res.Code)
	}

	if res := performRequest(app, http.MethodGet, "/api/v1/me", "", "bad-token"); res.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid token status 401, got %d", res.Code)
	}
}

func testRouter(t *testing.T) http.Handler {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}

	return router.New(router.Dependencies{
		Config: config.Config{
			AppEnv:             "test",
			JWTSecret:          "test-secret",
			CORSAllowedOrigins: []string{"http://localhost:5173"},
		},
		DB: db,
	})
}

func performRequest(handler http.Handler, method string, path string, body string, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}
