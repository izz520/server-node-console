package handler

import (
	"net/http"
	"time"

	"server-sing-box-2/backend/internal/domain"

	"github.com/gin-gonic/gin"
)

type adminUserResponse struct {
	ID        uint            `json:"id"`
	Username  string          `json:"username"`
	Email     string          `json:"email"`
	Role      domain.UserRole `json:"role"`
	CreatedAt time.Time       `json:"createdAt"`
}

func (h *Handler) AdminListUsers(c *gin.Context) {
	var users []domain.User
	if err := h.db.Order("created_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list users failed"})
		return
	}

	items := make([]adminUserResponse, 0, len(users))
	for _, user := range users {
		items = append(items, adminUserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) AdminListServers(c *gin.Context) {
	var servers []domain.Server
	if err := h.db.Order("created_at DESC").Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list servers failed"})
		return
	}

	items := make([]serverResponse, 0, len(servers))
	for _, server := range servers {
		items = append(items, toServerResponse(server))
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) AdminListNodes(c *gin.Context) {
	var nodes []domain.ProtocolNode
	if err := h.db.Order("created_at DESC").Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list nodes failed"})
		return
	}

	items := make([]nodeResponse, 0, len(nodes))
	for _, node := range nodes {
		items = append(items, toNodeResponse(node))
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) AdminListSubscriptions(c *gin.Context) {
	var subscriptions []domain.Subscription
	if err := h.db.Order("created_at DESC").Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list subscriptions failed"})
		return
	}

	items := make([]subscriptionResponse, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		response, err := h.toSubscriptionResponse(subscription, false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "build subscription response failed"})
			return
		}
		items = append(items, response)
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) AdminListTasks(c *gin.Context) {
	var tasks []domain.Task
	if err := h.db.Order("created_at DESC").Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list tasks failed"})
		return
	}

	items := make([]taskResponse, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, toTaskResponse(task))
	}
	c.JSON(http.StatusOK, items)
}
