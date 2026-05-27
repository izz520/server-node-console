package handler

import (
	"encoding/base64"
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

type nodeImportRequest struct {
	Mode        string `json:"mode" binding:"required"`
	Name        string `json:"name"`
	Protocol    string `json:"protocol"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
	ListenPort  int    `json:"listenPort"`
	PublicPort  *int   `json:"publicPort"`
	RawLink     string `json:"rawLink"`
	Remark      string `json:"remark"`
	ConfigJSON  string `json:"configJson"`
	Sensitive   string `json:"sensitive"`
	DisplayName string `json:"displayName"`
}

type nodeUpdateRequest struct {
	Name       string `json:"name" binding:"required,max=120"`
	Protocol   string `json:"protocol" binding:"required,max=120"`
	Address    string `json:"address" binding:"required,max=255"`
	Port       int    `json:"port" binding:"required,min=1,max=65535"`
	ListenPort int    `json:"listenPort" binding:"min=0,max=65535"`
	PublicPort *int   `json:"publicPort"`
	Remark     string `json:"remark"`
	ConfigJSON string `json:"configJson"`
	Sensitive  string `json:"sensitive"`
}

type nodeResponse struct {
	ID            uint                     `json:"id"`
	UserID        uint                     `json:"userId"`
	Name          string                   `json:"name"`
	ServerID      *uint                    `json:"serverId"`
	Protocol      string                   `json:"protocol"`
	Address       string                   `json:"address"`
	Port          int                      `json:"port"`
	ListenPort    int                      `json:"listenPort"`
	PublicPort    *int                     `json:"publicPort"`
	Remark        string                   `json:"remark"`
	InstallMethod domain.NodeInstallMethod `json:"installMethod"`
	Status        domain.NodeStatus        `json:"status"`
	HasSensitive  bool                     `json:"hasSensitive"`
	CreatedAt     time.Time                `json:"createdAt"`
	UpdatedAt     time.Time                `json:"updatedAt"`
}

type nodeConfig struct {
	Address    string `json:"address"`
	Port       int    `json:"port"`
	Remark     string `json:"remark,omitempty"`
	RawLink    string `json:"rawLink,omitempty"`
	ConfigJSON string `json:"configJson,omitempty"`
}

type encryptedNodeConfig struct {
	Sensitive string `json:"sensitive,omitempty"`
	RawLink   string `json:"rawLink,omitempty"`
}

func (h *Handler) ListNodes(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var nodes []domain.ProtocolNode
	if err := h.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list nodes failed"})
		return
	}

	items := make([]nodeResponse, 0, len(nodes))
	for _, node := range nodes {
		items = append(items, toNodeResponse(node))
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) ImportNode(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req nodeImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	normalized, encrypted, err := h.buildExternalNodePayload(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node := domain.ProtocolNode{
		UserID:                 userID,
		Name:                   normalized.Name,
		Protocol:               normalized.Protocol,
		ListenPort:             normalized.ListenPort,
		PublicPort:             normalized.PublicPort,
		EncryptedProtocolJSON:  encrypted,
		SubscriptionConfigJSON: normalized.SubscriptionConfigJSON(),
		InstallMethod:          domain.NodeInstallMethodExternal,
		Status:                 domain.NodeStatusImported,
	}
	if err := h.db.Create(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create node failed"})
		return
	}

	h.logOperation(&userID, "node.import", "node", map[string]any{"nodeId": node.ID, "name": node.Name, "mode": req.Mode})
	c.JSON(http.StatusCreated, toNodeResponse(node))
}

func (h *Handler) UpdateNode(c *gin.Context) {
	node, ok := h.findOwnedNode(c)
	if !ok {
		return
	}

	var req nodeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	normalized := normalizedNodePayload{
		Name:       strings.TrimSpace(req.Name),
		Protocol:   strings.TrimSpace(req.Protocol),
		Address:    strings.TrimSpace(req.Address),
		Port:       req.Port,
		ListenPort: req.ListenPort,
		PublicPort: req.PublicPort,
		Remark:     strings.TrimSpace(req.Remark),
		ConfigJSON: strings.TrimSpace(req.ConfigJSON),
	}
	if normalized.ListenPort == 0 {
		normalized.ListenPort = normalized.Port
	}
	if err := normalized.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if node.InstallMethod == domain.NodeInstallMethodSystem {
		h.updateSystemNodeSubscriptionSettings(c, node, normalized)
		return
	}

	encrypted, err := h.encryptNodeConfig(encryptedNodeConfig{
		Sensitive: strings.TrimSpace(req.Sensitive),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt node config failed"})
		return
	}

	node.Name = normalized.Name
	node.Protocol = normalized.Protocol
	node.ListenPort = normalized.ListenPort
	node.PublicPort = normalized.PublicPort
	if encrypted != "" {
		node.EncryptedProtocolJSON = encrypted
	}
	node.SubscriptionConfigJSON = normalized.SubscriptionConfigJSON()

	if err := h.db.Save(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update node failed"})
		return
	}

	h.logOperation(&node.UserID, "node.update", "node", map[string]any{"nodeId": node.ID, "name": node.Name})
	c.JSON(http.StatusOK, toNodeResponse(node))
}

func (h *Handler) updateSystemNodeSubscriptionSettings(c *gin.Context, node domain.ProtocolNode, normalized normalizedNodePayload) {
	config := nodeConfig{}
	_ = json.Unmarshal([]byte(node.SubscriptionConfigJSON), &config)
	listenPort := normalized.ListenPort
	if listenPort == 0 {
		listenPort = normalized.Port
	}
	if normalized.Protocol != node.Protocol ||
		normalized.Address != config.Address ||
		normalized.Port != config.Port ||
		listenPort != node.ListenPort {
		c.JSON(http.StatusBadRequest, gin.H{"error": "system installed node core settings cannot be edited"})
		return
	}

	config.Remark = normalized.Remark
	node.Name = normalized.Name
	node.PublicPort = normalized.PublicPort
	node.SubscriptionConfigJSON = updateNodeConfigJSON(node.SubscriptionConfigJSON, config)

	if err := h.db.Save(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update node failed"})
		return
	}

	h.logOperation(&node.UserID, "node.update", "node", map[string]any{"nodeId": node.ID, "name": node.Name})
	c.JSON(http.StatusOK, toNodeResponse(node))
}

func updateNodeConfigJSON(current string, config nodeConfig) string {
	values := map[string]any{}
	_ = json.Unmarshal([]byte(current), &values)
	values["address"] = config.Address
	values["port"] = config.Port
	if config.Remark == "" {
		delete(values, "remark")
	} else {
		values["remark"] = config.Remark
	}
	if config.RawLink == "" {
		delete(values, "rawLink")
	} else {
		values["rawLink"] = config.RawLink
	}
	if config.ConfigJSON == "" {
		delete(values, "configJson")
	} else {
		values["configJson"] = config.ConfigJSON
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (h *Handler) DeleteNode(c *gin.Context) {
	node, ok := h.findOwnedNode(c)
	if !ok {
		return
	}
	if node.InstallMethod == domain.NodeInstallMethodSystem &&
		node.Status != domain.NodeStatusUninstalled &&
		node.Status != domain.NodeStatusInstallFailed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "system installed node must be uninstalled before deletion"})
		return
	}

	if err := h.deleteNodeRecord(node.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete node failed"})
		return
	}

	h.logOperation(&node.UserID, "node.delete", "node", map[string]any{"nodeId": node.ID, "name": node.Name})
	c.Status(http.StatusNoContent)
}

func (h *Handler) deleteNodeRecord(nodeID uint) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("node_id = ?", nodeID).Delete(&domain.SubscriptionNode{}).Error; err != nil {
			return err
		}
		return tx.Delete(&domain.ProtocolNode{}, nodeID).Error
	})
}

func (h *Handler) findOwnedNode(c *gin.Context) (domain.ProtocolNode, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		return domain.ProtocolNode{}, false
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node id"})
		return domain.ProtocolNode{}, false
	}

	var node domain.ProtocolNode
	err = h.db.Where("id = ? AND user_id = ?", id, userID).First(&node).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return domain.ProtocolNode{}, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get node failed"})
		return domain.ProtocolNode{}, false
	}

	return node, true
}

type normalizedNodePayload struct {
	Name       string
	Protocol   string
	Address    string
	Port       int
	ListenPort int
	PublicPort *int
	Remark     string
	RawLink    string
	ConfigJSON string
	Sensitive  string
}

func (p normalizedNodePayload) Validate() error {
	if p.Name == "" {
		return errors.New("node name is required")
	}
	if p.Protocol == "" {
		return errors.New("node protocol is required")
	}
	if p.Address == "" {
		return errors.New("node address is required")
	}
	if p.Port < 1 || p.Port > 65535 {
		return errors.New("node port must be between 1 and 65535")
	}
	if p.ListenPort < 1 || p.ListenPort > 65535 {
		return errors.New("node listen port must be between 1 and 65535")
	}
	if p.PublicPort != nil && (*p.PublicPort < 1 || *p.PublicPort > 65535) {
		return errors.New("node public port must be between 1 and 65535")
	}
	return nil
}

func (p normalizedNodePayload) SubscriptionConfigJSON() string {
	config := nodeConfig{
		Address:    p.Address,
		Port:       p.Port,
		Remark:     p.Remark,
		RawLink:    p.RawLink,
		ConfigJSON: p.ConfigJSON,
	}
	data, err := json.Marshal(config)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (h *Handler) buildExternalNodePayload(req nodeImportRequest) (normalizedNodePayload, string, error) {
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	switch mode {
	case "manual":
		payload := normalizedNodePayload{
			Name:       strings.TrimSpace(req.Name),
			Protocol:   strings.TrimSpace(req.Protocol),
			Address:    strings.TrimSpace(req.Address),
			Port:       req.Port,
			ListenPort: req.ListenPort,
			PublicPort: req.PublicPort,
			Remark:     strings.TrimSpace(req.Remark),
			ConfigJSON: strings.TrimSpace(req.ConfigJSON),
		}
		if payload.ListenPort == 0 {
			payload.ListenPort = payload.Port
		}
		if err := payload.Validate(); err != nil {
			return normalizedNodePayload{}, "", err
		}
		encrypted, err := h.encryptNodeConfig(encryptedNodeConfig{
			Sensitive: strings.TrimSpace(req.Sensitive),
		})
		return payload, encrypted, err
	case "link":
		payload, err := parseShareLink(strings.TrimSpace(req.RawLink), strings.TrimSpace(req.DisplayName))
		if err != nil {
			return normalizedNodePayload{}, "", err
		}
		encrypted, err := h.encryptNodeConfig(encryptedNodeConfig{
			Sensitive: payload.Sensitive,
			RawLink:   payload.RawLink,
		})
		return payload, encrypted, err
	default:
		return normalizedNodePayload{}, "", errors.New("unsupported import mode")
	}
}

func (h *Handler) encryptNodeConfig(config encryptedNodeConfig) (string, error) {
	if config.Sensitive == "" && config.RawLink == "" {
		return "", nil
	}
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	encryptor, err := security.NewEncryptor(h.encryptionKey)
	if err != nil {
		return "", err
	}
	return encryptor.Encrypt(string(data))
}

func parseShareLink(rawLink string, displayName string) (normalizedNodePayload, error) {
	if rawLink == "" {
		return normalizedNodePayload{}, errors.New("share link is required")
	}

	if strings.HasPrefix(rawLink, "vmess://") {
		return parseVMessLink(rawLink, displayName)
	}
	if strings.HasPrefix(strings.ToLower(rawLink), "ss://") {
		if payload, err := parseSSLink(rawLink, displayName); err == nil {
			return payload, nil
		}
	}

	parsed, err := url.Parse(rawLink)
	if err != nil || parsed.Scheme == "" {
		return normalizedNodePayload{}, errors.New("invalid share link")
	}

	protocol := normalizeProtocol(parsed.Scheme)
	name := strings.TrimSpace(displayName)
	if name == "" {
		if fragment, err := url.QueryUnescape(parsed.Fragment); err == nil {
			name = strings.TrimSpace(fragment)
		}
	}
	if name == "" {
		name = fmt.Sprintf("%s-%s", protocol, parsed.Hostname())
	}

	port := 0
	if parsed.Port() != "" {
		port, _ = strconv.Atoi(parsed.Port())
	}
	if port == 0 {
		return normalizedNodePayload{}, errors.New("share link port is required")
	}

	payload := normalizedNodePayload{
		Name:       name,
		Protocol:   protocol,
		Address:    parsed.Hostname(),
		Port:       port,
		ListenPort: port,
		RawLink:    rawLink,
	}
	if protocol == "Vless" {
		payload.Sensitive = parseVLESSSensitive(parsed)
	}
	if err := payload.Validate(); err != nil {
		return normalizedNodePayload{}, err
	}
	return payload, nil
}

func parseVLESSSensitive(parsed *url.URL) string {
	values := map[string]string{}
	if parsed.User != nil {
		if uuid := strings.TrimSpace(parsed.User.Username()); uuid != "" {
			values["uuid"] = uuid
		}
	}
	query := parsed.Query()
	for _, key := range []string{"type", "security", "pbk", "fp", "sni", "sid", "spx", "flow", "encryption"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			values[key] = value
		}
	}
	if len(values) == 0 {
		return ""
	}
	data, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return string(data)
}

func parseSSLink(rawLink string, displayName string) (normalizedNodePayload, error) {
	withoutScheme := strings.TrimPrefix(rawLink, "ss://")
	mainPart := withoutScheme
	if index := strings.IndexAny(mainPart, "?#"); index >= 0 {
		mainPart = mainPart[:index]
	}
	if !strings.Contains(mainPart, "@") {
		decoded := decodeBase64String(mainPart)
		if decoded == "" {
			return normalizedNodePayload{}, errors.New("invalid shadowsocks link")
		}
		rawLink = "ss://" + decoded
		if fragmentIndex := strings.Index(withoutScheme, "#"); fragmentIndex >= 0 {
			rawLink += withoutScheme[fragmentIndex:]
		}
	}

	parsed, err := url.Parse(rawLink)
	if err != nil || parsed.Hostname() == "" {
		return normalizedNodePayload{}, errors.New("invalid shadowsocks link")
	}
	port := 0
	if parsed.Port() != "" {
		port, _ = strconv.Atoi(parsed.Port())
	}
	if port == 0 {
		return normalizedNodePayload{}, errors.New("share link port is required")
	}
	name := strings.TrimSpace(displayName)
	if name == "" {
		if fragment, err := url.QueryUnescape(parsed.Fragment); err == nil {
			name = strings.TrimSpace(fragment)
		}
	}
	if name == "" {
		name = "ss-" + parsed.Hostname()
	}
	payload := normalizedNodePayload{
		Name:       name,
		Protocol:   "Shadowsocks-2022",
		Address:    parsed.Hostname(),
		Port:       port,
		ListenPort: port,
		RawLink:    rawLink,
	}
	if err := payload.Validate(); err != nil {
		return normalizedNodePayload{}, err
	}
	return payload, nil
}

func parseVMessLink(rawLink string, displayName string) (normalizedNodePayload, error) {
	encoded := strings.TrimPrefix(rawLink, "vmess://")
	decoded, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(encoded)
	}
	if err != nil {
		return normalizedNodePayload{}, errors.New("invalid vmess link")
	}

	var payload map[string]any
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return normalizedNodePayload{}, errors.New("invalid vmess payload")
	}

	address, _ := payload["add"].(string)
	name := strings.TrimSpace(displayName)
	if name == "" {
		name, _ = payload["ps"].(string)
	}
	if name == "" {
		name = "vmess-" + address
	}

	port := parseAnyInt(payload["port"])
	normalized := normalizedNodePayload{
		Name:       name,
		Protocol:   "Vmess-ws",
		Address:    address,
		Port:       port,
		ListenPort: port,
		RawLink:    rawLink,
	}
	if err := normalized.Validate(); err != nil {
		return normalizedNodePayload{}, err
	}
	return normalized, nil
}

func parseAnyInt(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(typed)
		return parsed
	default:
		return 0
	}
}

func normalizeProtocol(scheme string) string {
	switch strings.ToLower(scheme) {
	case "vless":
		return "Vless"
	case "trojan":
		return "Trojan"
	case "ss":
		return "Shadowsocks-2022"
	case "hysteria2", "hy2":
		return "Hysteria2"
	case "tuic":
		return "Tuic"
	case "anytls":
		return "AnyTLS"
	case "socks", "socks5":
		return "Socks5"
	default:
		return scheme
	}
}

func toNodeResponse(node domain.ProtocolNode) nodeResponse {
	config := nodeConfig{}
	_ = json.Unmarshal([]byte(node.SubscriptionConfigJSON), &config)

	return nodeResponse{
		ID:            node.ID,
		UserID:        node.UserID,
		Name:          node.Name,
		ServerID:      node.ServerID,
		Protocol:      node.Protocol,
		Address:       config.Address,
		Port:          config.Port,
		ListenPort:    node.ListenPort,
		PublicPort:    node.PublicPort,
		Remark:        config.Remark,
		InstallMethod: node.InstallMethod,
		Status:        node.Status,
		HasSensitive:  node.EncryptedProtocolJSON != "",
		CreatedAt:     node.CreatedAt,
		UpdatedAt:     node.UpdatedAt,
	}
}
