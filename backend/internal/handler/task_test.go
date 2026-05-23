package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"server-sing-box-2/backend/internal/domain"
)

type taskListResponse struct {
	ID     uint   `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

type taskDetailResponse struct {
	Task taskListResponse `json:"task"`
	Logs []struct {
		ID      uint   `json:"id"`
		Level   string `json:"level"`
		Message string `json:"message"`
	} `json:"logs"`
}

func TestTaskListAndDetail(t *testing.T) {
	app := testRouter(t)
	token := registerTestUser(t, app, "task-user", "task-user@example.com")
	userID := currentUserIDFromAPI(t, app, token)
	createTestTask(t, userID, "install started")

	listRes := performRequest(app, http.MethodGet, "/api/v1/tasks", "", token)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d: %s", listRes.Code, listRes.Body.String())
	}

	var list []taskListResponse
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode task list: %v", err)
	}
	if len(list) != 1 || list[0].Type != string(domain.TaskTypeInstall) {
		t.Fatalf("unexpected task list: %+v", list)
	}

	detailRes := performRequest(app, http.MethodGet, "/api/v1/tasks/1", "", token)
	if detailRes.Code != http.StatusOK {
		t.Fatalf("expected detail status 200, got %d: %s", detailRes.Code, detailRes.Body.String())
	}

	var detail taskDetailResponse
	if err := json.Unmarshal(detailRes.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode task detail: %v", err)
	}
	if len(detail.Logs) != 1 || detail.Logs[0].Message != "install started" {
		t.Fatalf("unexpected task detail: %+v", detail)
	}
}

func TestTaskRejectsCrossUserAccess(t *testing.T) {
	app := testRouter(t)
	ownerToken := registerTestUser(t, app, "task-owner", "task-owner@example.com")
	otherToken := registerTestUser(t, app, "task-other", "task-other@example.com")
	ownerID := currentUserIDFromAPI(t, app, ownerToken)
	createTestTask(t, ownerID, "owner task")

	listRes := performRequest(app, http.MethodGet, "/api/v1/tasks", "", otherToken)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected other list status 200, got %d", listRes.Code)
	}
	var list []taskListResponse
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode other task list: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected other user to see zero tasks, got %d", len(list))
	}

	detailRes := performRequest(app, http.MethodGet, "/api/v1/tasks/1", "", otherToken)
	if detailRes.Code != http.StatusNotFound {
		t.Fatalf("expected cross-user detail status 404, got %d", detailRes.Code)
	}
}

func TestAdminTaskDetail(t *testing.T) {
	app := testRouter(t)
	adminToken := registerTestUser(t, app, "task-admin", "task-admin@example.com")
	userToken := registerTestUser(t, app, "task-normal", "task-normal@example.com")
	userID := currentUserIDFromAPI(t, app, userToken)
	taskID := createTestTask(t, userID, "admin visible log")

	db := extractDB(t, app)
	if err := db.Model(&domain.User{}).Where("username = ?", "task-admin").Update("role", domain.UserRoleAdmin).Error; err != nil {
		t.Fatalf("promote admin: %v", err)
	}
	loginRes := performRequest(app, http.MethodPost, "/api/v1/auth/login", `{"account":"task-admin","password":"password123"}`, "")
	if loginRes.Code != http.StatusOK {
		t.Fatalf("admin login failed: %d %s", loginRes.Code, loginRes.Body.String())
	}
	var auth authResponse
	if err := json.Unmarshal(loginRes.Body.Bytes(), &auth); err != nil {
		t.Fatalf("decode admin login: %v", err)
	}
	adminToken = auth.Token

	forbiddenRes := performRequest(app, http.MethodGet, "/api/v1/admin/tasks/"+strconvUint(taskID), "", userToken)
	if forbiddenRes.Code != http.StatusForbidden {
		t.Fatalf("expected normal user admin task detail status 403, got %d", forbiddenRes.Code)
	}

	adminRes := performRequest(app, http.MethodGet, "/api/v1/admin/tasks/"+strconvUint(taskID), "", adminToken)
	if adminRes.Code != http.StatusOK {
		t.Fatalf("expected admin task detail status 200, got %d: %s", adminRes.Code, adminRes.Body.String())
	}

	var detail taskDetailResponse
	if err := json.Unmarshal(adminRes.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode admin task detail: %v", err)
	}
	if len(detail.Logs) != 1 || detail.Logs[0].Message != "admin visible log" {
		t.Fatalf("unexpected admin task detail: %+v", detail)
	}
}

func createTestTask(t *testing.T, userID uint, message string) uint {
	t.Helper()

	db := extractDB(t, nil)
	task := domain.Task{
		UserID: userID,
		Type:   domain.TaskTypeInstall,
		Status: domain.TaskStatusRunning,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := db.Create(&domain.TaskLog{
		TaskID:  task.ID,
		Level:   "info",
		Message: message,
	}).Error; err != nil {
		t.Fatalf("create task log: %v", err)
	}
	return task.ID
}

func currentUserIDFromAPI(t *testing.T, app http.Handler, token string) uint {
	t.Helper()

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
	return me.ID
}
