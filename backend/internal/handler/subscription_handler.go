package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"server-sing-box-2/backend/internal/domain"
	"server-sing-box-2/backend/internal/security"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type subscriptionRequest struct {
	Name    string                    `json:"name" binding:"required,max=120"`
	Format  domain.SubscriptionFormat `json:"format" binding:"required"`
	Enabled bool                      `json:"enabled"`
	NodeIDs []uint                    `json:"nodeIds" binding:"required"`
	Remark  string                    `json:"remark"`
}

type subscriptionResponse struct {
	ID              uint                      `json:"id"`
	UserID          uint                      `json:"userId"`
	Name            string                    `json:"name"`
	Enabled         bool                      `json:"enabled"`
	Format          domain.SubscriptionFormat `json:"format"`
	NodeIDs         []uint                    `json:"nodeIds"`
	NodeCount       int                       `json:"nodeCount"`
	Token           string                    `json:"token,omitempty"`
	SubscriptionURL string                    `json:"subscriptionUrl,omitempty"`
	Remark          string                    `json:"remark"`
	CreatedAt       time.Time                 `json:"createdAt"`
	UpdatedAt       time.Time                 `json:"updatedAt"`
}

type subscriptionNodeView struct {
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	Address    string `json:"address"`
	Port       int    `json:"port"`
	Remark     string `json:"remark,omitempty"`
	RawLink    string `json:"rawLink,omitempty"`
	ConfigJSON string `json:"configJson,omitempty"`
}

func (h *Handler) ListSubscriptions(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var subscriptions []domain.Subscription
	if err := h.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list subscriptions failed"})
		return
	}

	items := make([]subscriptionResponse, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		response, err := h.toSubscriptionResponse(subscription, true)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "build subscription response failed"})
			return
		}
		items = append(items, response)
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) GetSubscription(c *gin.Context) {
	subscription, ok := h.findOwnedSubscription(c)
	if !ok {
		return
	}

	response, err := h.toSubscriptionResponse(subscription, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "build subscription response failed"})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) CreateSubscription(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req subscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}
	req = normalizeSubscriptionRequest(req)
	if err := h.validateSubscriptionNodes(userID, req.NodeIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, tokenHash, encryptedToken, err := h.newSubscriptionToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate subscription token failed"})
		return
	}

	subscription := domain.Subscription{
		UserID:         userID,
		Name:           req.Name,
		TokenHash:      tokenHash,
		EncryptedToken: encryptedToken,
		Enabled:        req.Enabled,
		Format:         req.Format,
		Remark:         req.Remark,
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&subscription).Error; err != nil {
			return err
		}
		return replaceSubscriptionNodes(tx, subscription.ID, req.NodeIDs)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create subscription failed"})
		return
	}

	response, err := h.toSubscriptionResponse(subscription, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "build subscription response failed"})
		return
	}
	response.Token = token
	response.SubscriptionURL = "/sub/" + token
	h.logOperation(&userID, "subscription.create", "subscription", map[string]any{"subscriptionId": subscription.ID, "name": subscription.Name})
	c.JSON(http.StatusCreated, response)
}

func (h *Handler) UpdateSubscription(c *gin.Context) {
	subscription, ok := h.findOwnedSubscription(c)
	if !ok {
		return
	}

	var req subscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}
	req = normalizeSubscriptionRequest(req)
	if err := h.validateSubscriptionNodes(subscription.UserID, req.NodeIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subscription.Name = req.Name
	subscription.Enabled = req.Enabled
	subscription.Format = req.Format
	subscription.Remark = req.Remark

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&subscription).Error; err != nil {
			return err
		}
		return replaceSubscriptionNodes(tx, subscription.ID, req.NodeIDs)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update subscription failed"})
		return
	}

	response, err := h.toSubscriptionResponse(subscription, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "build subscription response failed"})
		return
	}
	h.logOperation(&subscription.UserID, "subscription.update", "subscription", map[string]any{"subscriptionId": subscription.ID, "name": subscription.Name})
	c.JSON(http.StatusOK, response)
}

func (h *Handler) DeleteSubscription(c *gin.Context) {
	subscription, ok := h.findOwnedSubscription(c)
	if !ok {
		return
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("subscription_id = ?", subscription.ID).Delete(&domain.SubscriptionNode{}).Error; err != nil {
			return err
		}
		return tx.Delete(&subscription).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete subscription failed"})
		return
	}

	h.logOperation(&subscription.UserID, "subscription.delete", "subscription", map[string]any{"subscriptionId": subscription.ID, "name": subscription.Name})
	c.Status(http.StatusNoContent)
}

func (h *Handler) ResetSubscriptionToken(c *gin.Context) {
	subscription, ok := h.findOwnedSubscription(c)
	if !ok {
		return
	}

	token, tokenHash, encryptedToken, err := h.newSubscriptionToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate subscription token failed"})
		return
	}

	subscription.TokenHash = tokenHash
	subscription.EncryptedToken = encryptedToken
	if err := h.db.Save(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "reset subscription token failed"})
		return
	}

	response, err := h.toSubscriptionResponse(subscription, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "build subscription response failed"})
		return
	}
	response.Token = token
	response.SubscriptionURL = "/sub/" + token
	h.logOperation(&subscription.UserID, "subscription.reset_token", "subscription", map[string]any{"subscriptionId": subscription.ID, "name": subscription.Name})
	c.JSON(http.StatusOK, response)
}

func (h *Handler) PublicSubscription(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	var subscription domain.Subscription
	err := h.db.Where("token_hash = ?", hashToken(token)).First(&subscription).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get subscription failed"})
		return
	}
	if !subscription.Enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "subscription disabled"})
		return
	}

	nodes, err := h.subscriptionNodeViews(subscription.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "build subscription failed"})
		return
	}

	content, contentType, err := renderSubscription(subscription.Format, nodes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, contentType, []byte(content))
}

func (h *Handler) findOwnedSubscription(c *gin.Context) (domain.Subscription, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		return domain.Subscription{}, false
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return domain.Subscription{}, false
	}

	var subscription domain.Subscription
	err = h.db.Where("id = ? AND user_id = ?", id, userID).First(&subscription).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return domain.Subscription{}, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get subscription failed"})
		return domain.Subscription{}, false
	}

	return subscription, true
}

func (h *Handler) validateSubscriptionNodes(userID uint, nodeIDs []uint) error {
	if len(nodeIDs) == 0 {
		return errors.New("subscription must include at least one node")
	}
	uniqueIDs := uniqueUint(nodeIDs)
	var count int64
	if err := h.db.Model(&domain.ProtocolNode{}).
		Where("user_id = ? AND id IN ? AND status IN ?", userID, uniqueIDs, []domain.NodeStatus{domain.NodeStatusImported, domain.NodeStatusInstallOK}).
		Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(uniqueIDs)) {
		return errors.New("one or more nodes are invalid")
	}
	return nil
}

func (h *Handler) newSubscriptionToken() (string, string, string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	encryptor, err := security.NewEncryptor(h.encryptionKey)
	if err != nil {
		return "", "", "", err
	}
	encryptedToken, err := encryptor.Encrypt(token)
	if err != nil {
		return "", "", "", err
	}
	return token, hashToken(token), encryptedToken, nil
}

func (h *Handler) toSubscriptionResponse(subscription domain.Subscription, includeToken bool) (subscriptionResponse, error) {
	nodeIDs, err := h.subscriptionNodeIDs(subscription.ID)
	if err != nil {
		return subscriptionResponse{}, err
	}

	response := subscriptionResponse{
		ID:        subscription.ID,
		UserID:    subscription.UserID,
		Name:      subscription.Name,
		Enabled:   subscription.Enabled,
		Format:    subscription.Format,
		NodeIDs:   nodeIDs,
		NodeCount: len(nodeIDs),
		Remark:    subscription.Remark,
		CreatedAt: subscription.CreatedAt,
		UpdatedAt: subscription.UpdatedAt,
	}

	if includeToken && subscription.EncryptedToken != "" {
		encryptor, err := security.NewEncryptor(h.encryptionKey)
		if err != nil {
			return subscriptionResponse{}, err
		}
		token, err := encryptor.Decrypt(subscription.EncryptedToken)
		if err != nil {
			return subscriptionResponse{}, err
		}
		response.Token = token
		response.SubscriptionURL = "/sub/" + token
	}

	return response, nil
}

func (h *Handler) subscriptionNodeIDs(subscriptionID uint) ([]uint, error) {
	var links []domain.SubscriptionNode
	if err := h.db.Where("subscription_id = ?", subscriptionID).Order("sort_order ASC").Find(&links).Error; err != nil {
		return nil, err
	}
	nodeIDs := make([]uint, 0, len(links))
	for _, link := range links {
		nodeIDs = append(nodeIDs, link.NodeID)
	}
	return nodeIDs, nil
}

func (h *Handler) subscriptionNodeViews(subscriptionID uint) ([]subscriptionNodeView, error) {
	var links []domain.SubscriptionNode
	if err := h.db.Where("subscription_id = ?", subscriptionID).Order("sort_order ASC").Find(&links).Error; err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return []subscriptionNodeView{}, nil
	}

	nodeIDs := make([]uint, 0, len(links))
	for _, link := range links {
		nodeIDs = append(nodeIDs, link.NodeID)
	}

	var nodes []domain.ProtocolNode
	if err := h.db.Where("id IN ? AND status IN ?", nodeIDs, []domain.NodeStatus{domain.NodeStatusImported, domain.NodeStatusInstallOK}).Find(&nodes).Error; err != nil {
		return nil, err
	}

	byID := make(map[uint]domain.ProtocolNode, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}

	views := make([]subscriptionNodeView, 0, len(links))
	for _, link := range links {
		node, ok := byID[link.NodeID]
		if !ok {
			continue
		}
		config := nodeConfig{}
		_ = json.Unmarshal([]byte(node.SubscriptionConfigJSON), &config)
		port := config.Port
		if node.PublicPort != nil {
			port = *node.PublicPort
		}
		views = append(views, subscriptionNodeView{
			Name:       node.Name,
			Protocol:   node.Protocol,
			Address:    config.Address,
			Port:       port,
			Remark:     config.Remark,
			RawLink:    config.RawLink,
			ConfigJSON: config.ConfigJSON,
		})
	}
	return views, nil
}

func replaceSubscriptionNodes(tx *gorm.DB, subscriptionID uint, nodeIDs []uint) error {
	if err := tx.Where("subscription_id = ?", subscriptionID).Delete(&domain.SubscriptionNode{}).Error; err != nil {
		return err
	}
	for index, nodeID := range uniqueUint(nodeIDs) {
		if err := tx.Create(&domain.SubscriptionNode{
			SubscriptionID: subscriptionID,
			NodeID:         nodeID,
			SortOrder:      index,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeSubscriptionRequest(req subscriptionRequest) subscriptionRequest {
	req.Name = strings.TrimSpace(req.Name)
	req.Remark = strings.TrimSpace(req.Remark)
	req.NodeIDs = uniqueUint(req.NodeIDs)
	return req
}

func uniqueUint(values []uint) []uint {
	seen := make(map[uint]struct{}, len(values))
	out := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func renderSubscription(format domain.SubscriptionFormat, nodes []subscriptionNodeView) (string, string, error) {
	switch format {
	case domain.SubscriptionFormatSingBox:
		outbounds := make([]map[string]any, 0, len(nodes))
		for _, node := range nodes {
			outbounds = append(outbounds, node.toSingBoxOutbound())
		}
		data, _ := json.MarshalIndent(map[string]any{"outbounds": outbounds}, "", "  ")
		return string(data), "application/json; charset=utf-8", nil
	case domain.SubscriptionFormatClashMihomo:
		lines := []string{"proxies:"}
		for _, node := range nodes {
			lines = append(lines, fmt.Sprintf("  - name: %q", node.Name))
			lines = append(lines, fmt.Sprintf("    type: %q", node.clashType()))
			lines = append(lines, fmt.Sprintf("    server: %q", node.Address))
			lines = append(lines, fmt.Sprintf("    port: %d", node.Port))
		}
		return strings.Join(lines, "\n") + "\n", "text/yaml; charset=utf-8", nil
	case domain.SubscriptionFormatV2RayN, domain.SubscriptionFormatShadowrocket:
		lines := make([]string, 0, len(nodes))
		for _, node := range nodes {
			lines = append(lines, node.shareLine())
		}
		return strings.Join(lines, "\n"), "text/plain; charset=utf-8", nil
	case domain.SubscriptionFormatBase64:
		lines := make([]string, 0, len(nodes))
		for _, node := range nodes {
			lines = append(lines, node.shareLine())
		}
		return base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n"))), "text/plain; charset=utf-8", nil
	default:
		return "", "", errors.New("unsupported subscription format")
	}
}

func (node subscriptionNodeView) toSingBoxOutbound() map[string]any {
	if strings.TrimSpace(node.ConfigJSON) != "" {
		var config map[string]any
		if err := json.Unmarshal([]byte(node.ConfigJSON), &config); err == nil {
			if _, ok := config["tag"]; !ok {
				config["tag"] = node.Name
			}
			return config
		}
	}

	return map[string]any{
		"type":        node.singBoxType(),
		"tag":         node.Name,
		"server":      node.Address,
		"server_port": node.Port,
	}
}

func (node subscriptionNodeView) shareLine() string {
	if strings.TrimSpace(node.RawLink) != "" {
		return strings.TrimSpace(node.RawLink)
	}
	return fmt.Sprintf("%s://%s:%d#%s", node.shareScheme(), node.Address, node.Port, urlQueryEscape(node.Name))
}

func (node subscriptionNodeView) singBoxType() string {
	switch normalizedProtocol(node.Protocol) {
	case "shadowsocks":
		return "shadowsocks"
	case "hysteria2":
		return "hysteria2"
	case "tuic":
		return "tuic"
	case "socks":
		return "socks"
	case "vmess":
		return "vmess"
	case "vless":
		return "vless"
	case "anytls":
		return "anytls"
	default:
		return normalizedProtocol(node.Protocol)
	}
}

func (node subscriptionNodeView) clashType() string {
	switch normalizedProtocol(node.Protocol) {
	case "shadowsocks":
		return "ss"
	case "hysteria2":
		return "hysteria2"
	case "tuic":
		return "tuic"
	case "socks":
		return "socks5"
	case "vmess":
		return "vmess"
	case "vless":
		return "vless"
	default:
		return normalizedProtocol(node.Protocol)
	}
}

func (node subscriptionNodeView) shareScheme() string {
	switch normalizedProtocol(node.Protocol) {
	case "shadowsocks":
		return "ss"
	case "hysteria2":
		return "hysteria2"
	case "socks":
		return "socks5"
	default:
		return normalizedProtocol(node.Protocol)
	}
}

func normalizedProtocol(protocol string) string {
	value := strings.ToLower(strings.TrimSpace(protocol))
	switch {
	case strings.HasPrefix(value, "shadowsocks"):
		return "shadowsocks"
	case strings.HasPrefix(value, "hysteria2") || value == "hy2":
		return "hysteria2"
	case strings.HasPrefix(value, "tuic"):
		return "tuic"
	case strings.HasPrefix(value, "socks"):
		return "socks"
	case strings.HasPrefix(value, "vmess"):
		return "vmess"
	case strings.HasPrefix(value, "vless") || strings.Contains(value, "reality"):
		return "vless"
	case strings.HasPrefix(value, "anytls"):
		return "anytls"
	default:
		return strings.ReplaceAll(value, " ", "-")
	}
}

func urlQueryEscape(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}
