package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"server-sing-box-2/backend/internal/domain"

	"github.com/gin-gonic/gin"
)

type operationLogResponse struct {
	ID        uint      `json:"id"`
	UserID    *uint     `json:"userId"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Metadata  string    `json:"metadata"`
	CreatedAt time.Time `json:"createdAt"`
}

func (h *Handler) ListOperationLogs(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var logs []domain.OperationLog
	if err := h.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(200).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list operation logs failed"})
		return
	}
	c.JSON(http.StatusOK, toOperationLogResponses(logs))
}

func (h *Handler) AdminListOperationLogs(c *gin.Context) {
	var logs []domain.OperationLog
	if err := h.db.Order("created_at DESC").Limit(500).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list operation logs failed"})
		return
	}
	c.JSON(http.StatusOK, toOperationLogResponses(logs))
}

func (h *Handler) logOperation(userID *uint, action string, resource string, metadata map[string]any) {
	data, err := json.Marshal(metadata)
	if err != nil {
		data = []byte("{}")
	}
	_ = h.db.Create(&domain.OperationLog{
		UserID:   userID,
		Action:   action,
		Resource: resource,
		Metadata: string(data),
	}).Error
}

func toOperationLogResponses(logs []domain.OperationLog) []operationLogResponse {
	items := make([]operationLogResponse, 0, len(logs))
	for _, log := range logs {
		items = append(items, operationLogResponse{
			ID:        log.ID,
			UserID:    log.UserID,
			Action:    log.Action,
			Resource:  log.Resource,
			Metadata:  log.Metadata,
			CreatedAt: log.CreatedAt,
		})
	}
	return items
}
