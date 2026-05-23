package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"server-sing-box-2/backend/internal/domain"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type natMappingRequest struct {
	Name       string `json:"name" binding:"required,max=120"`
	Transport  string `json:"transport" binding:"max=16"`
	ListenPort int    `json:"listenPort" binding:"required,min=1,max=65535"`
	PublicPort int    `json:"publicPort" binding:"required,min=1,max=65535"`
	Remark     string `json:"remark"`
}

type natMappingResponse struct {
	ID         uint      `json:"id"`
	ServerID   uint      `json:"serverId"`
	Name       string    `json:"name"`
	Transport  string    `json:"transport"`
	ListenPort int       `json:"listenPort"`
	PublicPort int       `json:"publicPort"`
	Remark     string    `json:"remark"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (h *Handler) ListNATMappings(c *gin.Context) {
	userID, serverID, ok := h.resolveOwnedServerID(c)
	if !ok {
		return
	}

	var mappings []domain.NATPortMapping
	if err := h.db.Where("user_id = ? AND server_id = ?", userID, serverID).
		Order("created_at DESC").
		Find(&mappings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list nat mappings failed"})
		return
	}

	items := make([]natMappingResponse, 0, len(mappings))
	for _, mapping := range mappings {
		items = append(items, toNATMappingResponse(mapping))
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) CreateNATMapping(c *gin.Context) {
	userID, serverID, ok := h.resolveOwnedServerID(c)
	if !ok {
		return
	}

	var req natMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}
	req = normalizeNATMappingRequest(req)

	mapping := domain.NATPortMapping{
		UserID:     userID,
		ServerID:   serverID,
		Name:       req.Name,
		Transport:  req.Transport,
		ListenPort: req.ListenPort,
		PublicPort: req.PublicPort,
		Remark:     req.Remark,
	}
	if err := h.db.Create(&mapping).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create nat mapping failed"})
		return
	}

	h.logOperation(&userID, "nat_mapping.create", "nat_mapping", map[string]any{"mappingId": mapping.ID, "serverId": serverID})
	c.JSON(http.StatusCreated, toNATMappingResponse(mapping))
}

func (h *Handler) UpdateNATMapping(c *gin.Context) {
	mapping, ok := h.findOwnedNATMapping(c)
	if !ok {
		return
	}

	var req natMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}
	req = normalizeNATMappingRequest(req)

	mapping.Name = req.Name
	mapping.Transport = req.Transport
	mapping.ListenPort = req.ListenPort
	mapping.PublicPort = req.PublicPort
	mapping.Remark = req.Remark

	if err := h.db.Save(&mapping).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update nat mapping failed"})
		return
	}

	h.logOperation(&mapping.UserID, "nat_mapping.update", "nat_mapping", map[string]any{"mappingId": mapping.ID, "serverId": mapping.ServerID})
	c.JSON(http.StatusOK, toNATMappingResponse(mapping))
}

func (h *Handler) DeleteNATMapping(c *gin.Context) {
	mapping, ok := h.findOwnedNATMapping(c)
	if !ok {
		return
	}

	if err := h.db.Delete(&mapping).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete nat mapping failed"})
		return
	}

	h.logOperation(&mapping.UserID, "nat_mapping.delete", "nat_mapping", map[string]any{"mappingId": mapping.ID, "serverId": mapping.ServerID})
	c.Status(http.StatusNoContent)
}

func (h *Handler) resolveOwnedServerID(c *gin.Context) (uint, uint, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		return 0, 0, false
	}

	serverID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid server id"})
		return 0, 0, false
	}

	var count int64
	if err := h.db.Model(&domain.Server{}).
		Where("id = ? AND user_id = ?", serverID, userID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "check server ownership failed"})
		return 0, 0, false
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return 0, 0, false
	}

	return userID, uint(serverID), true
}

func (h *Handler) findOwnedNATMapping(c *gin.Context) (domain.NATPortMapping, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		return domain.NATPortMapping{}, false
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid nat mapping id"})
		return domain.NATPortMapping{}, false
	}

	var mapping domain.NATPortMapping
	err = h.db.Where("id = ? AND user_id = ?", id, userID).First(&mapping).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "nat mapping not found"})
		return domain.NATPortMapping{}, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get nat mapping failed"})
		return domain.NATPortMapping{}, false
	}

	return mapping, true
}

func normalizeNATMappingRequest(req natMappingRequest) natMappingRequest {
	req.Name = strings.TrimSpace(req.Name)
	req.Transport = strings.ToUpper(strings.TrimSpace(req.Transport))
	req.Remark = strings.TrimSpace(req.Remark)
	return req
}

func toNATMappingResponse(mapping domain.NATPortMapping) natMappingResponse {
	return natMappingResponse{
		ID:         mapping.ID,
		ServerID:   mapping.ServerID,
		Name:       mapping.Name,
		Transport:  mapping.Transport,
		ListenPort: mapping.ListenPort,
		PublicPort: mapping.PublicPort,
		Remark:     mapping.Remark,
		CreatedAt:  mapping.CreatedAt,
		UpdatedAt:  mapping.UpdatedAt,
	}
}
