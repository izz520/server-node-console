package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"server-sing-box-2/backend/internal/domain"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type clashTemplateRequest struct {
	Name    string `json:"name" binding:"required,max=120"`
	Content string `json:"content" binding:"required"`
	Remark  string `json:"remark"`
}

func (h *Handler) ListClashTemplates(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var templates []domain.ClashTemplate
	if err := h.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&templates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list clash templates failed"})
		return
	}
	c.JSON(http.StatusOK, templates)
}

func (h *Handler) CreateClashTemplate(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req clashTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}
	req = normalizeClashTemplateRequest(req)
	if err := validateClashTemplateContent(req.Content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template := domain.ClashTemplate{
		UserID:  userID,
		Name:    req.Name,
		Content: req.Content,
		Remark:  req.Remark,
	}
	if err := h.db.Create(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create clash template failed"})
		return
	}

	h.logOperation(&userID, "clash_template.create", "clash_template", map[string]any{"templateId": template.ID, "name": template.Name})
	c.JSON(http.StatusCreated, template)
}

func (h *Handler) UpdateClashTemplate(c *gin.Context) {
	template, ok := h.findOwnedClashTemplate(c)
	if !ok {
		return
	}

	var req clashTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}
	req = normalizeClashTemplateRequest(req)
	if err := validateClashTemplateContent(req.Content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template.Name = req.Name
	template.Content = req.Content
	template.Remark = req.Remark
	if err := h.db.Save(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update clash template failed"})
		return
	}

	h.logOperation(&template.UserID, "clash_template.update", "clash_template", map[string]any{"templateId": template.ID, "name": template.Name})
	c.JSON(http.StatusOK, template)
}

func (h *Handler) DeleteClashTemplate(c *gin.Context) {
	template, ok := h.findOwnedClashTemplate(c)
	if !ok {
		return
	}

	var count int64
	if err := h.db.Model(&domain.Subscription{}).
		Where("clash_template_id = ?", template.ID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "check clash template usage failed"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clash template is used by subscriptions"})
		return
	}

	if err := h.db.Delete(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete clash template failed"})
		return
	}

	h.logOperation(&template.UserID, "clash_template.delete", "clash_template", map[string]any{"templateId": template.ID, "name": template.Name})
	c.Status(http.StatusNoContent)
}

func (h *Handler) findOwnedClashTemplate(c *gin.Context) (domain.ClashTemplate, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		return domain.ClashTemplate{}, false
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid clash template id"})
		return domain.ClashTemplate{}, false
	}

	var template domain.ClashTemplate
	err = h.db.Where("id = ? AND user_id = ?", id, userID).First(&template).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "clash template not found"})
		return domain.ClashTemplate{}, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get clash template failed"})
		return domain.ClashTemplate{}, false
	}
	return template, true
}

func normalizeClashTemplateRequest(req clashTemplateRequest) clashTemplateRequest {
	req.Name = strings.TrimSpace(req.Name)
	req.Content = strings.TrimSpace(req.Content)
	req.Remark = strings.TrimSpace(req.Remark)
	return req
}

func validateClashTemplateContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return errors.New("clash template content is required")
	}
	if !strings.Contains(content, "proxies:") {
		return errors.New("clash template must include proxies")
	}
	return nil
}
