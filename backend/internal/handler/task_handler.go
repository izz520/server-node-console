package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"server-sing-box-2/backend/internal/domain"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type taskResponse struct {
	ID        uint              `json:"id"`
	UserID    uint              `json:"userId"`
	ServerID  *uint             `json:"serverId"`
	NodeID    *uint             `json:"nodeId"`
	Type      domain.TaskType   `json:"type"`
	Status    domain.TaskStatus `json:"status"`
	Error     string            `json:"error"`
	StartedAt *time.Time        `json:"startedAt"`
	EndedAt   *time.Time        `json:"endedAt"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

type taskLogResponse struct {
	ID        uint      `json:"id"`
	TaskID    uint      `json:"taskId"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type taskDetailResponse struct {
	Task taskResponse      `json:"task"`
	Logs []taskLogResponse `json:"logs"`
}

func (h *Handler) ListTasks(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var tasks []domain.Task
	if err := h.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list tasks failed"})
		return
	}

	items := make([]taskResponse, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, toTaskResponse(task))
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) GetTask(c *gin.Context) {
	task, ok := h.findOwnedTask(c)
	if !ok {
		return
	}

	h.respondTaskDetail(c, task)
}

func (h *Handler) AdminGetTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var task domain.Task
	err = h.db.First(&task, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get task failed"})
		return
	}

	h.respondTaskDetail(c, task)
}

func (h *Handler) respondTaskDetail(c *gin.Context, task domain.Task) {
	var logs []domain.TaskLog
	if err := h.db.Where("task_id = ?", task.ID).Order("created_at ASC").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list task logs failed"})
		return
	}

	items := make([]taskLogResponse, 0, len(logs))
	for _, log := range logs {
		items = append(items, toTaskLogResponse(log))
	}
	c.JSON(http.StatusOK, taskDetailResponse{
		Task: toTaskResponse(task),
		Logs: items,
	})
}

func (h *Handler) findOwnedTask(c *gin.Context) (domain.Task, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		return domain.Task{}, false
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return domain.Task{}, false
	}

	var task domain.Task
	err = h.db.Where("id = ? AND user_id = ?", id, userID).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return domain.Task{}, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get task failed"})
		return domain.Task{}, false
	}

	return task, true
}

func toTaskResponse(task domain.Task) taskResponse {
	return taskResponse{
		ID:        task.ID,
		UserID:    task.UserID,
		ServerID:  task.ServerID,
		NodeID:    task.NodeID,
		Type:      task.Type,
		Status:    task.Status,
		Error:     task.Error,
		StartedAt: task.StartedAt,
		EndedAt:   task.EndedAt,
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
	}
}

func toTaskLogResponse(log domain.TaskLog) taskLogResponse {
	return taskLogResponse{
		ID:        log.ID,
		TaskID:    log.TaskID,
		Level:     log.Level,
		Message:   log.Message,
		CreatedAt: log.CreatedAt,
	}
}
