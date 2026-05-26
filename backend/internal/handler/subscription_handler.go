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
	Name            string                    `json:"name" binding:"required,max=120"`
	Format          domain.SubscriptionFormat `json:"format" binding:"required"`
	ClashTemplate   string                    `json:"clashTemplate"`
	ClashTemplateID *uint                     `json:"clashTemplateId"`
	Enabled         bool                      `json:"enabled"`
	NodeIDs         []uint                    `json:"nodeIds" binding:"required"`
	Remark          string                    `json:"remark"`
}

type subscriptionResponse struct {
	ID              uint                      `json:"id"`
	UserID          uint                      `json:"userId"`
	Name            string                    `json:"name"`
	Enabled         bool                      `json:"enabled"`
	Format          domain.SubscriptionFormat `json:"format"`
	ClashTemplate   string                    `json:"clashTemplate"`
	ClashTemplateID *uint                     `json:"clashTemplateId,omitempty"`
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
	UUID       string `json:"uuid,omitempty"`
	Password   string `json:"password,omitempty"`
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
	if err := h.validateClashTemplate(userID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, tokenHash, encryptedToken, err := h.newSubscriptionToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate subscription token failed"})
		return
	}

	subscription := domain.Subscription{
		UserID:          userID,
		Name:            req.Name,
		TokenHash:       tokenHash,
		EncryptedToken:  encryptedToken,
		Enabled:         req.Enabled,
		Format:          req.Format,
		ClashTemplate:   req.ClashTemplate,
		ClashTemplateID: req.ClashTemplateID,
		Remark:          req.Remark,
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
	if err := h.validateClashTemplate(subscription.UserID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subscription.Name = req.Name
	subscription.Enabled = req.Enabled
	subscription.Format = req.Format
	subscription.ClashTemplate = req.ClashTemplate
	subscription.ClashTemplateID = req.ClashTemplateID
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

	customClashTemplate, err := h.subscriptionClashTemplateContent(subscription)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load clash template failed"})
		return
	}
	content, contentType, err := renderSubscription(subscription.Format, subscription.ClashTemplate, customClashTemplate, nodes)
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

func (h *Handler) validateClashTemplate(userID uint, req subscriptionRequest) error {
	if req.Format != domain.SubscriptionFormatClashMihomo || req.ClashTemplate != "custom" {
		return nil
	}
	if req.ClashTemplateID == nil || *req.ClashTemplateID == 0 {
		return errors.New("clash template is required")
	}
	var count int64
	if err := h.db.Model(&domain.ClashTemplate{}).
		Where("id = ? AND user_id = ?", *req.ClashTemplateID, userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("clash template not found")
	}
	return nil
}

func (h *Handler) subscriptionClashTemplateContent(subscription domain.Subscription) (string, error) {
	if subscription.Format != domain.SubscriptionFormatClashMihomo ||
		normalizeClashTemplate(subscription.ClashTemplate) != "custom" ||
		subscription.ClashTemplateID == nil {
		return "", nil
	}
	var template domain.ClashTemplate
	if err := h.db.Where("id = ? AND user_id = ?", *subscription.ClashTemplateID, subscription.UserID).First(&template).Error; err != nil {
		return "", err
	}
	return template.Content, nil
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
		ID:              subscription.ID,
		UserID:          subscription.UserID,
		Name:            subscription.Name,
		Enabled:         subscription.Enabled,
		Format:          subscription.Format,
		ClashTemplate:   normalizeClashTemplate(subscription.ClashTemplate),
		ClashTemplateID: subscription.ClashTemplateID,
		NodeIDs:         nodeIDs,
		NodeCount:       len(nodeIDs),
		Remark:          subscription.Remark,
		CreatedAt:       subscription.CreatedAt,
		UpdatedAt:       subscription.UpdatedAt,
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
	sensitiveByNodeID, err := h.subscriptionNodeSensitiveValues(nodes)
	if err != nil {
		return nil, err
	}
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
		sensitiveValues := sensitiveByNodeID[node.ID]
		views = append(views, subscriptionNodeView{
			Name:       node.Name,
			Protocol:   node.Protocol,
			Address:    config.Address,
			Port:       port,
			Remark:     config.Remark,
			RawLink:    config.RawLink,
			ConfigJSON: config.ConfigJSON,
			UUID:       sensitiveValues["uuid"],
			Password:   firstNonEmpty(sensitiveValues["password"], sensitiveValues["passwd"], sensitiveValues["pass"], sensitiveValues["uuid"]),
		})
	}
	return views, nil
}

func (h *Handler) subscriptionNodeSensitiveValues(nodes []domain.ProtocolNode) (map[uint]map[string]string, error) {
	out := make(map[uint]map[string]string, len(nodes))
	var encryptor *security.Encryptor

	for _, node := range nodes {
		if strings.TrimSpace(node.EncryptedProtocolJSON) == "" {
			continue
		}
		if encryptor == nil {
			var err error
			encryptor, err = security.NewEncryptor(h.encryptionKey)
			if err != nil {
				return nil, err
			}
		}
		plain, err := encryptor.Decrypt(node.EncryptedProtocolJSON)
		if err != nil {
			return nil, err
		}
		encryptedConfig := encryptedNodeConfig{}
		if err := json.Unmarshal([]byte(plain), &encryptedConfig); err != nil {
			return nil, err
		}
		if strings.TrimSpace(encryptedConfig.Sensitive) == "" {
			continue
		}
		out[node.ID] = parseSensitiveValues(encryptedConfig.Sensitive)
	}

	return out, nil
}

func parseSensitiveValues(value string) map[string]string {
	values := map[string]string{}
	if err := json.Unmarshal([]byte(value), &values); err == nil {
		return values
	}
	for _, line := range strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ';' || r == '&'
	}) {
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			key, raw, ok = strings.Cut(line, ":")
		}
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		raw = strings.TrimSpace(raw)
		if key == "" || raw == "" {
			continue
		}
		values[key] = raw
	}
	return values
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
	req.ClashTemplate = normalizeClashTemplate(req.ClashTemplate)
	if req.Format != domain.SubscriptionFormatClashMihomo || req.ClashTemplate != "custom" {
		req.ClashTemplateID = nil
	}
	req.Remark = strings.TrimSpace(req.Remark)
	req.NodeIDs = uniqueUint(req.NodeIDs)
	return req
}

func normalizeClashTemplate(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "rule-cn", "balanced":
		return "rule-cn"
	case "global-proxy", "global":
		return "global-proxy"
	case "custom":
		return "custom"
	default:
		return "rule-cn"
	}
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

func renderSubscription(format domain.SubscriptionFormat, clashTemplate string, customClashTemplate string, nodes []subscriptionNodeView) (string, string, error) {
	switch format {
	case domain.SubscriptionFormatSingBox:
		outbounds := make([]map[string]any, 0, len(nodes))
		for _, node := range nodes {
			outbounds = append(outbounds, node.toSingBoxOutbound())
		}
		data, _ := json.MarshalIndent(map[string]any{"outbounds": outbounds}, "", "  ")
		return string(data), "application/json; charset=utf-8", nil
	case domain.SubscriptionFormatClashMihomo:
		return renderClashMihomo(clashTemplate, customClashTemplate, nodes), "text/yaml; charset=utf-8", nil
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

func renderClashMihomo(template string, customTemplate string, nodes []subscriptionNodeView) string {
	template = normalizeClashTemplate(template)
	if template == "custom" && strings.TrimSpace(customTemplate) != "" {
		return renderCustomClashMihomo(customTemplate, nodes)
	}
	proxyNames := make([]string, 0, len(nodes))
	for _, node := range nodes {
		proxyNames = append(proxyNames, node.Name)
	}

	lines := []string{
		"mixed-port: 7890",
		"allow-lan: false",
		"mode: " + clashModeForTemplate(template),
		"log-level: info",
		"ipv6: false",
		"external-controller: 127.0.0.1:9090",
		"",
		"dns:",
		"  enable: true",
		"  listen: 0.0.0.0:1053",
		"  enhanced-mode: fake-ip",
		"  fake-ip-range: 198.18.0.1/16",
		"  default-nameserver:",
		"    - 223.5.5.5",
		"    - 119.29.29.29",
		"  nameserver:",
		"    - https://dns.alidns.com/dns-query",
		"    - https://doh.pub/dns-query",
		"  fallback:",
		"    - https://1.1.1.1/dns-query",
		"    - https://8.8.8.8/dns-query",
		"",
		"proxies:",
	}
	if len(nodes) == 0 {
		lines = append(lines, "  []")
	} else {
		for _, node := range nodes {
			lines = append(lines, node.clashProxyLines()...)
		}
	}

	lines = append(lines,
		"",
		"proxy-groups:",
		"  - name: "+yamlQuote("PROXY"),
		"    type: select",
	)
	if len(proxyNames) == 0 {
		lines = append(lines, "    proxies:", "      - DIRECT")
	} else {
		lines = append(lines, "    proxies:")
		for _, name := range proxyNames {
			lines = append(lines, "      - "+yamlQuote(name))
		}
		lines = append(lines, "      - DIRECT")
	}
	lines = append(lines,
		"  - name: "+yamlQuote("AUTO"),
		"    type: url-test",
		"    url: http://www.gstatic.com/generate_204",
		"    interval: 300",
		"    tolerance: 50",
	)
	if len(proxyNames) == 0 {
		lines = append(lines, "    proxies:", "      - DIRECT")
	} else {
		lines = append(lines, "    proxies:")
		for _, name := range proxyNames {
			lines = append(lines, "      - "+yamlQuote(name))
		}
	}
	lines = append(lines, "", "rules:")
	lines = append(lines, clashRulesForTemplate(template)...)

	return strings.Join(lines, "\n") + "\n"
}

func renderCustomClashMihomo(template string, nodes []subscriptionNodeView) string {
	lines := splitYAMLLines(template)
	lines = replaceTopLevelYAMLBlock(lines, "proxies:", clashProxyBlock(nodes))
	lines = replaceProxyGroupProxyLists(lines, clashProxyNames(nodes))
	return strings.Join(lines, "\n") + "\n"
}

func clashProxyBlock(nodes []subscriptionNodeView) []string {
	if len(nodes) == 0 {
		return []string{"  []"}
	}
	lines := make([]string, 0, len(nodes)*6)
	for _, node := range nodes {
		lines = append(lines, node.clashProxyLines()...)
	}
	return lines
}

func clashProxyNames(nodes []subscriptionNodeView) []string {
	names := make([]string, 0, len(nodes))
	for _, node := range nodes {
		names = append(names, node.Name)
	}
	return names
}

func splitYAMLLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.TrimRight(value, "\n")
	if value == "" {
		return []string{}
	}
	return strings.Split(value, "\n")
}

func replaceTopLevelYAMLBlock(lines []string, key string, block []string) []string {
	start := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == key && leadingSpaces(line) == 0 {
			start = index
			break
		}
	}
	if start == -1 {
		out := append([]string{}, lines...)
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, strings.TrimSuffix(key, ":")+":")
		out = append(out, block...)
		return out
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		line := lines[index]
		if strings.TrimSpace(line) == "" {
			continue
		}
		if leadingSpaces(line) == 0 && !strings.HasPrefix(strings.TrimSpace(line), "-") {
			end = index
			break
		}
	}
	out := make([]string, 0, len(lines)-end+start+1+len(block))
	out = append(out, lines[:start+1]...)
	out = append(out, block...)
	out = append(out, lines[end:]...)
	return out
}

func replaceProxyGroupProxyLists(lines []string, proxyNames []string) []string {
	groupStart := findTopLevelYAMLKey(lines, "proxy-groups:")
	if groupStart == -1 {
		return lines
	}
	groupEnd := len(lines)
	for index := groupStart + 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) != "" && leadingSpaces(lines[index]) == 0 {
			groupEnd = index
			break
		}
	}

	out := make([]string, 0, len(lines)+len(proxyNames)*2)
	out = append(out, lines[:groupStart+1]...)
	for index := groupStart + 1; index < groupEnd; index++ {
		line := lines[index]
		out = append(out, line)
		if strings.TrimSpace(line) != "proxies:" {
			continue
		}
		indent := leadingSpaces(line)
		for index+1 < groupEnd {
			next := lines[index+1]
			if strings.TrimSpace(next) != "" && leadingSpaces(next) <= indent {
				break
			}
			index++
		}
		if len(proxyNames) == 0 {
			out = append(out, strings.Repeat(" ", indent+2)+"- DIRECT")
			continue
		}
		for _, name := range proxyNames {
			out = append(out, strings.Repeat(" ", indent+2)+"- "+yamlQuote(name))
		}
		out = append(out, strings.Repeat(" ", indent+2)+"- DIRECT")
	}
	out = append(out, lines[groupEnd:]...)
	return out
}

func findTopLevelYAMLKey(lines []string, key string) int {
	for index, line := range lines {
		if strings.TrimSpace(line) == key && leadingSpaces(line) == 0 {
			return index
		}
	}
	return -1
}

func leadingSpaces(value string) int {
	return len(value) - len(strings.TrimLeft(value, " "))
}

func clashModeForTemplate(template string) string {
	if normalizeClashTemplate(template) == "global-proxy" {
		return "global"
	}
	return "rule"
}

func clashRulesForTemplate(template string) []string {
	if normalizeClashTemplate(template) == "global-proxy" {
		return []string{
			"  - GEOIP,LAN,DIRECT",
			"  - MATCH,PROXY",
		}
	}
	return []string{
		"  - GEOIP,LAN,DIRECT",
		"  - GEOIP,CN,DIRECT",
		"  - MATCH,PROXY",
	}
}

func (node subscriptionNodeView) clashProxyLines() []string {
	lines := []string{
		"  - name: " + yamlQuote(node.Name),
		"    type: " + yamlQuote(node.clashType()),
		"    server: " + yamlQuote(node.Address),
		fmt.Sprintf("    port: %d", node.Port),
	}
	switch normalizedProtocol(node.Protocol) {
	case "anytls":
		if strings.TrimSpace(node.Password) != "" {
			lines = append(lines, "    password: "+yamlQuote(strings.TrimSpace(node.Password)))
		}
		lines = append(lines, "    skip-cert-verify: true")
	case "hysteria2", "tuic":
		if strings.TrimSpace(node.Password) != "" {
			lines = append(lines, "    password: "+yamlQuote(strings.TrimSpace(node.Password)))
		}
		lines = append(lines, "    skip-cert-verify: true")
	case "vless", "vmess":
		if strings.TrimSpace(node.UUID) != "" {
			lines = append(lines, "    uuid: "+yamlQuote(strings.TrimSpace(node.UUID)))
		}
		lines = append(lines, "    tls: true", "    skip-cert-verify: true")
	}
	return lines
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
	if normalizedProtocol(node.Protocol) == "anytls" && strings.TrimSpace(node.UUID) != "" {
		return fmt.Sprintf("%s://%s@%s:%d?insecure=1&allowInsecure=1#%s", node.shareScheme(), url.QueryEscape(strings.TrimSpace(node.UUID)), node.Address, node.Port, urlQueryEscape(node.Name))
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

func yamlQuote(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(data)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
