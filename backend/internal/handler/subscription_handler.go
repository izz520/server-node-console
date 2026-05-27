package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"server-sing-box-2/backend/internal/converter"
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
	Name          string `json:"name"`
	Protocol      string `json:"protocol"`
	Address       string `json:"address"`
	Port          int    `json:"port"`
	Remark        string `json:"remark,omitempty"`
	RawLink       string `json:"rawLink,omitempty"`
	ConfigJSON    string `json:"configJson,omitempty"`
	UUID          string `json:"uuid,omitempty"`
	Password      string `json:"password,omitempty"`
	Peer          string `json:"peer,omitempty"`
	HPKP          string `json:"hpkp,omitempty"`
	UDP           bool   `json:"udp,omitempty"`
	Insecure      bool   `json:"insecure,omitempty"`
	AllowInsecure bool   `json:"allowInsecure,omitempty"`
	Network       string `json:"network,omitempty"`
	Security      string `json:"security,omitempty"`
	Flow          string `json:"flow,omitempty"`
	Fingerprint   string `json:"fingerprint,omitempty"`
	ServerName    string `json:"serverName,omitempty"`
	RealityPBK    string `json:"realityPbk,omitempty"`
	RealitySID    string `json:"realitySid,omitempty"`
	SpiderX       string `json:"spiderX,omitempty"`
	Values        map[string]string
}

const encryptedRawLinkKey = "__rawLink"

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
	if err := h.loadChainProxyDependencies(byID, nodes); err != nil {
		return nil, err
	}

	allNodes := protocolNodesFromMap(byID)
	views := make([]subscriptionNodeView, 0, len(allNodes))
	sensitiveByNodeID, err := h.subscriptionNodeSensitiveValues(allNodes)
	if err != nil {
		return nil, err
	}
	included := map[uint]struct{}{}
	for _, link := range links {
		node, ok := byID[link.NodeID]
		if !ok {
			continue
		}
		views = append(views, subscriptionNodeViewFromNode(node, byID, sensitiveByNodeID[node.ID]))
		included[node.ID] = struct{}{}
	}
	for _, node := range allNodes {
		if _, ok := included[node.ID]; ok {
			continue
		}
		views = append(views, subscriptionNodeViewFromNode(node, byID, sensitiveByNodeID[node.ID]))
	}
	return views, nil
}

func protocolNodesFromMap(byID map[uint]domain.ProtocolNode) []domain.ProtocolNode {
	nodes := make([]domain.ProtocolNode, 0, len(byID))
	for _, node := range byID {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	return nodes
}

func subscriptionNodeViewFromNode(node domain.ProtocolNode, byID map[uint]domain.ProtocolNode, encryptedValues map[string]string) subscriptionNodeView {
	config := nodeConfig{}
	_ = json.Unmarshal([]byte(node.SubscriptionConfigJSON), &config)
	port := config.Port
	if node.PublicPort != nil {
		port = *node.PublicPort
	}
	rawLink := firstNonEmpty(config.RawLink, mapValue(encryptedValues, encryptedRawLinkKey))
	rawLinkValues := parseShareLinkValues(rawLink)
	sensitiveValues := mergeStringMaps(rawLinkValues, encryptedValues)
	if rawLink != "" {
		sensitiveValues = mergeStringMaps(encryptedValues, rawLinkValues)
		if node.PublicPort == nil && node.InstallMethod == domain.NodeInstallMethodSystem {
			if endpoint, ok := shareLinkEndpointFromRawLink(rawLink); ok {
				port = endpoint.Port
			}
		}
	}
	if config.ChainProxyNodeID != nil {
		if upstream, ok := byID[*config.ChainProxyNodeID]; ok {
			sensitiveValues["dialer-proxy"] = upstream.Name
		}
	}
	return subscriptionNodeView{
		Name:          node.Name,
		Protocol:      node.Protocol,
		Address:       config.Address,
		Port:          port,
		Remark:        config.Remark,
		RawLink:       rawLink,
		ConfigJSON:    config.ConfigJSON,
		UUID:          mapValue(sensitiveValues, "uuid", "id"),
		Password:      firstNonEmpty(mapValue(sensitiveValues, "password", "passwd", "pass"), mapValue(sensitiveValues, "uuid", "id")),
		Peer:          mapValue(sensitiveValues, "peer", "sni", "server_name", "servername", "serverName"),
		HPKP:          mapValue(sensitiveValues, "hpkp", "pin", "certificate_public_key_sha256"),
		UDP:           truthyValue(mapValue(sensitiveValues, "udp")),
		Insecure:      truthyValue(mapValue(sensitiveValues, "insecure", "skip-cert-verify", "skip_cert_verify", "allowInsecure", "allow_insecure")),
		AllowInsecure: truthyValue(mapValue(sensitiveValues, "allowInsecure", "allow_insecure")),
		Network:       mapValue(sensitiveValues, "network", "type"),
		Security:      mapValue(sensitiveValues, "security"),
		Flow:          mapValue(sensitiveValues, "flow"),
		Fingerprint:   mapValue(sensitiveValues, "fp", "fingerprint", "client-fingerprint", "client_fingerprint"),
		ServerName:    mapValue(sensitiveValues, "sni", "servername", "server_name", "serverName", "host"),
		RealityPBK:    mapValue(sensitiveValues, "pbk", "public-key", "public_key"),
		RealitySID:    mapValue(sensitiveValues, "sid", "short-id", "short_id"),
		SpiderX:       mapValue(sensitiveValues, "spx", "spider-x", "spider_x"),
		Values:        sensitiveValues,
	}
}

func (h *Handler) loadChainProxyDependencies(byID map[uint]domain.ProtocolNode, initial []domain.ProtocolNode) error {
	queue := append([]domain.ProtocolNode{}, initial...)
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		config := nodeConfig{}
		_ = json.Unmarshal([]byte(node.SubscriptionConfigJSON), &config)
		if config.ChainProxyNodeID == nil || *config.ChainProxyNodeID == 0 {
			continue
		}
		if _, ok := byID[*config.ChainProxyNodeID]; ok {
			continue
		}
		var upstream domain.ProtocolNode
		if err := h.db.Where("user_id = ? AND id = ? AND status IN ?", node.UserID, *config.ChainProxyNodeID, []domain.NodeStatus{domain.NodeStatusImported, domain.NodeStatusInstallOK}).First(&upstream).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		byID[upstream.ID] = upstream
		queue = append(queue, upstream)
	}
	return nil
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
		values := map[string]string{}
		if strings.TrimSpace(encryptedConfig.Sensitive) != "" {
			values = parseSensitiveValues(encryptedConfig.Sensitive)
		}
		if rawLink := strings.TrimSpace(encryptedConfig.RawLink); rawLink != "" {
			values[encryptedRawLinkKey] = rawLink
		}
		if len(values) > 0 {
			out[node.ID] = values
		}
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

func parseShareLinkValues(rawLink string) map[string]string {
	values := map[string]string{}
	rawLink = strings.TrimSpace(rawLink)
	if rawLink == "" {
		return values
	}
	if converted := converter.ParseKnownShareLinkValues(rawLink); len(converted) > 0 {
		values = mergeStringMaps(values, converted)
	}
	if strings.HasPrefix(rawLink, "vmess://") {
		encoded := strings.TrimPrefix(rawLink, "vmess://")
		decoded, err := base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			decoded, err = base64.StdEncoding.DecodeString(encoded)
		}
		if err != nil {
			return values
		}
		var payload map[string]any
		if err := json.Unmarshal(decoded, &payload); err != nil {
			return values
		}
		copyAnyString(values, payload, "uuid", "id")
		copyAnyString(values, payload, "alterId", "aid")
		copyAnyString(values, payload, "cipher", "scy")
		copyAnyString(values, payload, "network", "net")
		copyAnyString(values, payload, "tls", "tls")
		copyAnyString(values, payload, "servername", "sni")
		copyAnyString(values, payload, "host", "host")
		copyAnyString(values, payload, "path", "path")
		return values
	}
	if strings.HasPrefix(strings.ToLower(rawLink), "ss://") {
		withoutScheme := strings.TrimPrefix(rawLink, "ss://")
		mainPart := withoutScheme
		if index := strings.IndexAny(mainPart, "?#"); index >= 0 {
			mainPart = mainPart[:index]
		}
		if !strings.Contains(mainPart, "@") {
			if decoded := decodeBase64String(mainPart); decoded != "" {
				cipherPassword, _, _ := strings.Cut(decoded, "@")
				cipher, pass, ok := strings.Cut(cipherPassword, ":")
				if ok {
					values["cipher"] = cipher
					values["password"] = pass
				}
			}
		}
	}

	parsed, err := url.Parse(rawLink)
	if err != nil {
		return values
	}
	scheme := strings.ToLower(parsed.Scheme)
	if parsed.User != nil {
		username := strings.TrimSpace(parsed.User.Username())
		password, hasPassword := parsed.User.Password()
		password = strings.TrimSpace(password)
		switch scheme {
		case "ss":
			if hasPassword {
				values["cipher"] = username
				values["password"] = password
			} else if decoded := decodeBase64String(username); decoded != "" {
				cipher, pass, ok := strings.Cut(decoded, ":")
				if ok {
					values["cipher"] = cipher
					values["password"] = pass
				}
			}
		case "vless", "vmess":
			values["uuid"] = username
		case "trojan", "anytls", "hysteria", "hysteria2", "hy2":
			values["password"] = username
			if scheme == "anytls" {
				values["uuid"] = username
			}
		case "socks", "socks5":
			values["username"] = username
			if hasPassword {
				values["password"] = password
			}
		case "tuic":
			if hasPassword {
				values["uuid"] = username
				values["password"] = password
			} else {
				values["token"] = username
			}
		}
	}
	for key, items := range parsed.Query() {
		if len(items) == 0 {
			continue
		}
		if value := strings.TrimSpace(items[0]); value != "" {
			values[key] = value
		}
	}
	return values
}

func copyAnyString(out map[string]string, payload map[string]any, target string, source string) {
	value, ok := payload[source]
	if !ok {
		return
	}
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) != "" {
			out[target] = strings.TrimSpace(typed)
		}
	case float64:
		out[target] = strconv.Itoa(int(typed))
	case bool:
		out[target] = strconv.FormatBool(typed)
	}
}

func decodeBase64String(value string) string {
	if decoded, err := base64.RawURLEncoding.DecodeString(value); err == nil {
		return string(decoded)
	}
	if decoded, err := base64.URLEncoding.DecodeString(value); err == nil {
		return string(decoded)
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(value); err == nil {
		return string(decoded)
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return string(decoded)
	}
	return ""
}

func mergeStringMaps(base map[string]string, overrides map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(overrides))
	for key, value := range base {
		if strings.TrimSpace(value) != "" {
			out[key] = value
		}
	}
	for key, value := range overrides {
		if strings.TrimSpace(value) != "" {
			out[key] = value
		}
	}
	return out
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
		trimmedLine := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmedLine, "proxies:") {
			continue
		}
		proxiesValue := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "proxies:"))
		if proxiesValue != "" && !strings.HasPrefix(proxiesValue, "&") {
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
	case "shadowsocks":
		lines = appendClashField(lines, "cipher", mapValue(node.Values, "cipher", "method", "encryption"))
		lines = appendClashField(lines, "password", node.Password)
		lines = appendClashBoolField(lines, "udp", mapValue(node.Values, "udp"))
		lines = appendClashField(lines, "udp-over-tcp", mapValue(node.Values, "udp-over-tcp", "uot"))
		lines = appendClashField(lines, "udp-over-tcp-version", mapValue(node.Values, "udp-over-tcp-version", "uot-version"))
		lines = appendClashField(lines, "ip-version", mapValue(node.Values, "ip-version"))
		lines = appendClashField(lines, "plugin", mapValue(node.Values, "plugin"))
		lines = appendPluginOpts(lines, node.Values)
	case "anytls":
		if strings.TrimSpace(node.Password) != "" {
			lines = append(lines, "    password: "+yamlQuote(strings.TrimSpace(node.Password)))
		}
		lines = appendClashField(lines, "client-fingerprint", firstNonEmpty(node.Fingerprint, "firefox"))
		lines = appendClashBoolField(lines, "udp", mapValue(node.Values, "udp"))
		lines = appendClashField(lines, "idle-session-check-interval", firstNonEmpty(mapValue(node.Values, "idle-session-check-interval"), "30"))
		lines = appendClashField(lines, "idle-session-timeout", firstNonEmpty(mapValue(node.Values, "idle-session-timeout"), "30"))
		lines = appendClashField(lines, "min-idle-session", mapValue(node.Values, "min-idle-session"))
		lines = appendClashField(lines, "sni", firstNonEmpty(node.Peer, node.ServerName))
		lines = appendClashField(lines, "fingerprint", node.HPKP)
		lines = appendClashALPN(lines, mapValue(node.Values, "alpn"))
		lines = appendClashBool(lines, "skip-cert-verify", node.Insecure || node.AllowInsecure)
	case "trojan":
		lines = appendClashField(lines, "password", node.Password)
		lines = appendClashBoolField(lines, "udp", mapValue(node.Values, "udp"))
		lines = appendTLSFields(lines, node, "sni")
		lines = appendClashField(lines, "network", node.Network)
		lines = appendRealityOpts(lines, node)
		lines = appendClashBool(lines, "skip-cert-verify", node.Insecure || node.AllowInsecure)
	case "hysteria2":
		lines = appendClashField(lines, "ports", mapValue(node.Values, "ports"))
		lines = appendClashField(lines, "hop-interval", mapValue(node.Values, "hop-interval", "hop_interval"))
		lines = appendClashField(lines, "password", node.Password)
		lines = appendClashField(lines, "up", mapValue(node.Values, "up", "upmbps"))
		lines = appendClashField(lines, "down", mapValue(node.Values, "down", "downmbps"))
		lines = appendClashField(lines, "obfs", mapValue(node.Values, "obfs"))
		lines = appendClashField(lines, "obfs-password", mapValue(node.Values, "obfs-password", "obfs_password"))
		lines = appendTLSFields(lines, node, "sni")
		lines = appendClashBool(lines, "skip-cert-verify", node.Insecure || node.AllowInsecure)
	case "tuic":
		lines = appendClashField(lines, "token", mapValue(node.Values, "token"))
		lines = appendClashField(lines, "uuid", node.UUID)
		lines = appendClashField(lines, "password", node.Password)
		lines = appendClashField(lines, "ip", mapValue(node.Values, "ip"))
		lines = appendClashField(lines, "heartbeat-interval", mapValue(node.Values, "heartbeat-interval"))
		lines = appendClashALPN(lines, mapValue(node.Values, "alpn"))
		lines = appendClashBoolField(lines, "disable-sni", mapValue(node.Values, "disable-sni"))
		lines = appendClashBoolField(lines, "reduce-rtt", mapValue(node.Values, "reduce-rtt"))
		lines = appendClashField(lines, "request-timeout", mapValue(node.Values, "request-timeout"))
		lines = appendClashField(lines, "udp-relay-mode", mapValue(node.Values, "udp-relay-mode"))
		lines = appendClashField(lines, "congestion-controller", mapValue(node.Values, "congestion-controller"))
		lines = appendClashField(lines, "bbr-profile", mapValue(node.Values, "bbr-profile"))
		lines = appendClashField(lines, "max-udp-relay-packet-size", mapValue(node.Values, "max-udp-relay-packet-size"))
		lines = appendClashBoolField(lines, "fast-open", mapValue(node.Values, "fast-open"))
		lines = appendClashField(lines, "max-open-streams", mapValue(node.Values, "max-open-streams"))
		lines = appendClashField(lines, "sni", node.ServerName)
		lines = appendClashBool(lines, "skip-cert-verify", node.Insecure || node.AllowInsecure)
	case "hysteria":
		lines = appendClashField(lines, "auth-str", firstNonEmpty(mapValue(node.Values, "auth-str", "auth_str", "auth"), node.Password))
		lines = appendClashField(lines, "ports", mapValue(node.Values, "ports"))
		lines = appendClashField(lines, "obfs", mapValue(node.Values, "obfs"))
		lines = appendClashField(lines, "protocol", mapValue(node.Values, "protocol"))
		lines = appendClashField(lines, "up", mapValue(node.Values, "up", "upmbps"))
		lines = appendClashField(lines, "down", mapValue(node.Values, "down", "downmbps"))
		lines = appendTLSFields(lines, node, "sni")
		lines = appendClashField(lines, "recv-window-conn", mapValue(node.Values, "recv-window-conn"))
		lines = appendClashField(lines, "recv-window", mapValue(node.Values, "recv-window"))
		lines = appendClashBoolField(lines, "disable_mtu_discovery", mapValue(node.Values, "disable_mtu_discovery"))
		lines = appendClashBoolField(lines, "fast-open", mapValue(node.Values, "fast-open"))
		lines = appendClashBool(lines, "skip-cert-verify", node.Insecure || node.AllowInsecure)
	case "socks":
		if username := mapValue(node.Values, "username", "user"); username != "" {
			lines = appendClashField(lines, "username", username)
		}
		lines = appendClashField(lines, "password", node.Password)
		lines = appendClashBoolField(lines, "tls", mapValue(node.Values, "tls"))
		lines = appendClashBoolField(lines, "udp", mapValue(node.Values, "udp"))
	case "vless", "vmess":
		lines = appendClashField(lines, "uuid", node.UUID)
		if normalizedProtocol(node.Protocol) == "vmess" {
			lines = appendClashField(lines, "alterId", firstNonEmpty(mapValue(node.Values, "alterId", "alterid", "aid"), "0"))
			lines = appendClashField(lines, "cipher", firstNonEmpty(mapValue(node.Values, "cipher", "scy"), "auto"))
		}
		lines = appendClashField(lines, "flow", node.Flow)
		lines = appendClashField(lines, "packet-encoding", mapValue(node.Values, "packet-encoding", "packet_encoding"))
		lines = appendClashField(lines, "encryption", mapValue(node.Values, "encryption"))
		lines = appendClashBoolField(lines, "udp", mapValue(node.Values, "udp"))
		if truthyValue(mapValue(node.Values, "tls")) || strings.EqualFold(node.Security, "tls") || strings.EqualFold(node.Security, "reality") {
			lines = append(lines, "    tls: true")
		}
		lines = appendTLSFields(lines, node, "servername")
		lines = appendClashField(lines, "network", node.Network)
		lines = appendRealityOpts(lines, node)
		lines = appendTransportOpts(lines, node.Values)
		if mapValue(node.Values, "global-padding") != "" {
			lines = appendClashBoolField(lines, "global-padding", mapValue(node.Values, "global-padding"))
		}
		if mapValue(node.Values, "authenticated-length") != "" {
			lines = appendClashBoolField(lines, "authenticated-length", mapValue(node.Values, "authenticated-length"))
		}
		lines = appendClashBoolField(lines, "tfo", firstNonEmpty(mapValue(node.Values, "tfo"), "false"))
		lines = append(lines, fmt.Sprintf("    skip-cert-verify: %t", node.Insecure || node.AllowInsecure))
	}
	lines = appendCommonClashFields(lines, node.Values)
	return lines
}

func appendClashField(lines []string, key string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return lines
	}
	return append(lines, "    "+key+": "+yamlScalar(value))
}

func appendClashBool(lines []string, key string, value bool) []string {
	return append(lines, fmt.Sprintf("    %s: %t", key, value))
}

func appendClashBoolField(lines []string, key string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return lines
	}
	return append(lines, fmt.Sprintf("    %s: %t", key, truthyValue(value)))
}

func appendClashALPN(lines []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return lines
	}
	items := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '|'
	})
	lines = append(lines, "    alpn:")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			lines = append(lines, "      - "+yamlQuote(item))
		}
	}
	return lines
}

func appendTLSFields(lines []string, node subscriptionNodeView, serverNameKey string) []string {
	if serverName := strings.TrimSpace(node.ServerName); serverName != "" {
		lines = appendClashField(lines, serverNameKey, serverName)
	}
	lines = appendClashField(lines, "fingerprint", mapValue(node.Values, "fingerprint"))
	lines = appendClashField(lines, "client-fingerprint", node.Fingerprint)
	lines = appendClashALPN(lines, mapValue(node.Values, "alpn"))
	return lines
}

func appendRealityOpts(lines []string, node subscriptionNodeView) []string {
	if !strings.EqualFold(strings.TrimSpace(node.Security), "reality") && strings.TrimSpace(node.RealityPBK) == "" && strings.TrimSpace(node.RealitySID) == "" {
		return lines
	}
	lines = append(lines, "    reality-opts:")
	if publicKey := strings.TrimSpace(node.RealityPBK); publicKey != "" {
		lines = append(lines, "      public-key: "+yamlQuote(publicKey))
	}
	if shortID := strings.TrimSpace(node.RealitySID); shortID != "" {
		lines = append(lines, "      short-id: "+yamlQuote(shortID))
	}
	if support := mapValue(node.Values, "support-x25519mlkem768"); support != "" {
		lines = append(lines, fmt.Sprintf("      support-x25519mlkem768: %t", truthyValue(support)))
	}
	return lines
}

func appendTransportOpts(lines []string, values map[string]string) []string {
	switch strings.ToLower(strings.TrimSpace(mapValue(values, "network", "type"))) {
	case "ws":
		if path := mapValue(values, "path", "ws-path"); path != "" || mapValue(values, "host") != "" {
			lines = append(lines, "    ws-opts:")
			if path != "" {
				lines = append(lines, "      path: "+yamlQuote(path))
			}
			if host := mapValue(values, "host"); host != "" {
				lines = append(lines, "      headers:")
				lines = append(lines, "        Host: "+yamlQuote(host))
			}
		}
	case "grpc":
		if serviceName := mapValue(values, "serviceName", "service-name", "grpc-service-name"); serviceName != "" {
			lines = append(lines, "    grpc-opts:")
			lines = append(lines, "      grpc-service-name: "+yamlQuote(serviceName))
		}
	case "http", "h2":
		if host := mapValue(values, "host"); host != "" || mapValue(values, "path") != "" {
			lines = append(lines, "    h2-opts:")
			if host != "" {
				lines = append(lines, "      host:")
				for _, item := range strings.Split(host, ",") {
					if item = strings.TrimSpace(item); item != "" {
						lines = append(lines, "        - "+yamlQuote(item))
					}
				}
			}
			if path := mapValue(values, "path"); path != "" {
				lines = append(lines, "      path: "+yamlQuote(path))
			}
		}
	case "xhttp":
		if path := mapValue(values, "path"); path != "" || mapValue(values, "host") != "" || mapValue(values, "mode") != "" {
			lines = append(lines, "    xhttp-opts:")
			if path := mapValue(values, "path"); path != "" {
				lines = append(lines, "      path: "+yamlQuote(path))
			}
			if host := mapValue(values, "host"); host != "" {
				lines = append(lines, "      host: "+yamlQuote(host))
			}
			if mode := mapValue(values, "mode"); mode != "" {
				lines = append(lines, "      mode: "+yamlQuote(mode))
			}
		}
	}
	return lines
}

func appendPluginOpts(lines []string, values map[string]string) []string {
	mode := mapValue(values, "mode", "obfs")
	host := mapValue(values, "host", "obfs-host", "obfs_param")
	if mode == "" && host == "" {
		return lines
	}
	lines = append(lines, "    plugin-opts:")
	if mode != "" {
		lines = append(lines, "      mode: "+yamlQuote(mode))
	}
	if host != "" {
		lines = append(lines, "      host: "+yamlQuote(host))
	}
	return lines
}

func appendCommonClashFields(lines []string, values map[string]string) []string {
	lines = appendClashField(lines, "ip-version", mapValue(values, "ip-version"))
	lines = appendClashField(lines, "interface-name", mapValue(values, "interface-name"))
	lines = appendClashField(lines, "routing-mark", mapValue(values, "routing-mark"))
	lines = appendClashBoolField(lines, "mptcp", mapValue(values, "mptcp"))
	lines = appendClashField(lines, "dialer-proxy", mapValue(values, "dialer-proxy"))
	lines = appendUnknownClashFields(lines, values)
	return lines
}

func appendUnknownClashFields(lines []string, values map[string]string) []string {
	written := map[string]struct{}{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		key, _, ok := strings.Cut(trimmed, ":")
		if ok {
			written[key] = struct{}{}
		}
	}
	for _, key := range sortedStringKeys(values) {
		if !isSafeClashFieldKey(key) || isInternalNodeValueKey(key) {
			continue
		}
		if _, ok := written[key]; ok {
			continue
		}
		lines = appendClashField(lines, key, values[key])
	}
	return lines
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isSafeClashFieldKey(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func isInternalNodeValueKey(key string) bool {
	switch key {
	case encryptedRawLinkKey, "id", "uuid", "password", "passwd", "pass", "method", "type", "security", "address", "server", "add", "port", "name", "ps", "fp", "sni", "server_name", "serverName", "servername", "pbk", "public-key", "public_key", "sid", "short-id", "short_id", "spx", "spider-x", "spider_x", "peer", "hpkp", "pin", "certificate_public_key_sha256", "allowInsecure", "allow_insecure":
		return true
	default:
		return false
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
		return shareLinkWithEndpoint(strings.TrimSpace(node.RawLink), node.Address, node.Port)
	}
	if normalizedProtocol(node.Protocol) == "anytls" && strings.TrimSpace(node.UUID) != "" {
		return fmt.Sprintf("%s://%s@%s:%d%s#%s", node.shareScheme(), url.QueryEscape(strings.TrimSpace(node.UUID)), node.Address, node.Port, node.anyTLSQuery(), urlQueryEscape(node.Name))
	}
	return fmt.Sprintf("%s://%s:%d#%s", node.shareScheme(), node.Address, node.Port, urlQueryEscape(node.Name))
}

func shareLinkWithEndpoint(rawLink string, address string, port int) string {
	address = strings.TrimSpace(address)
	if rawLink == "" || address == "" || port <= 0 {
		return rawLink
	}
	parsed, err := url.Parse(rawLink)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return rawLink
	}
	if strings.EqualFold(parsed.Hostname(), address) && parsed.Port() == strconv.Itoa(port) {
		return rawLink
	}
	return strings.Replace(rawLink, parsed.Host, net.JoinHostPort(address, strconv.Itoa(port)), 1)
}

func (node subscriptionNodeView) anyTLSQuery() string {
	params := make([]string, 0, 5)
	if peer := strings.TrimSpace(node.Peer); peer != "" {
		params = append(params, "peer="+anyTLSParamEscape(peer))
	}
	if node.UDP {
		params = append(params, "udp=1")
	}
	if hpkp := strings.TrimSpace(node.HPKP); hpkp != "" {
		params = append(params, "hpkp="+anyTLSParamEscape(hpkp))
	}
	if node.Insecure || (len(params) == 0 && !node.AllowInsecure) {
		params = append(params, "insecure=1")
	}
	if node.AllowInsecure || (len(params) == 1 && params[0] == "insecure=1") {
		params = append(params, "allowInsecure=1")
	}
	if len(params) == 0 {
		return ""
	}
	return "?" + strings.Join(params, "&")
}

func (node subscriptionNodeView) singBoxType() string {
	switch normalizedProtocol(node.Protocol) {
	case "shadowsocks":
		return "shadowsocks"
	case "hysteria2":
		return "hysteria2"
	case "hysteria":
		return "hysteria"
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
	case strings.HasPrefix(value, "hysteria"):
		return "hysteria"
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

func anyTLSParamEscape(value string) string {
	return strings.ReplaceAll(urlQueryEscape(value), "%3A", ":")
}

func yamlQuote(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(data)
}

func yamlScalar(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return `""`
	}
	if truthyValue(value) || strings.EqualFold(value, "false") || looksNumeric(value) {
		return value
	}
	return yamlQuote(value)
}

func looksNumeric(value string) bool {
	if value == "" {
		return false
	}
	if _, err := strconv.Atoi(value); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return true
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func mapValue(values map[string]string, keys ...string) string {
	for _, key := range keys {
		for _, candidate := range []string{key, strings.ToLower(key), strings.ReplaceAll(key, "_", "-"), strings.ReplaceAll(key, "-", "_")} {
			if value := strings.TrimSpace(values[candidate]); value != "" {
				return value
			}
		}
	}
	return ""
}

func truthyValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
