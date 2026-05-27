package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"server-sing-box-2/backend/internal/argosbx"
	"server-sing-box-2/backend/internal/domain"
	"server-sing-box-2/backend/internal/security"
	"server-sing-box-2/backend/internal/sshclient"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type installNodeRequest struct {
	ServerID         uint   `json:"serverId" binding:"required"`
	Name             string `json:"name" binding:"required,max=120"`
	Protocol         string `json:"protocol" binding:"required,max=120"`
	Port             int    `json:"port" binding:"min=0,max=65535"`
	PublicPort       *int   `json:"publicPort"`
	UUID             string `json:"uuid"`
	RealityDomain    string `json:"realityDomain"`
	CDNDomain        string `json:"cdnDomain"`
	ArgoMode         string `json:"argoMode"`
	ArgoDomain       string `json:"argoDomain"`
	ArgoToken        string `json:"argoToken"`
	NamePrefix       string `json:"namePrefix"`
	Remark           string `json:"remark"`
	ChainProxyNodeID *uint  `json:"chainProxyNodeId"`
}

type uninstallNodeRequest struct {
	DeleteAfterUninstall bool `json:"deleteAfterUninstall"`
}

type installNodeResponse struct {
	Node nodeResponse `json:"node"`
	Task taskResponse `json:"task"`
}

type installConfig struct {
	Address          string `json:"address"`
	Port             int    `json:"port"`
	Remark           string `json:"remark,omitempty"`
	GeneratedFrom    string `json:"generatedFrom"`
	ChainProxyNodeID *uint  `json:"chainProxyNodeId,omitempty"`
}

type installTarget struct {
	NodeID uint
	Req    installNodeRequest
}

func (h *Handler) InstallNode(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req installNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	server, ok := h.findServerByID(c, userID, req.ServerID)
	if !ok {
		return
	}
	if server.Status != domain.ServerStatusNormal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server is not ready for installation"})
		return
	}

	req, err := prepareInstallRequest(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := argosbx.VarNameForProtocol(req.Protocol); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !h.validateChainProxyNode(c, userID, 0, req.ChainProxyNodeID) {
		return
	}

	configJSON, _ := json.Marshal(installConfig{
		Address:          server.Host,
		Port:             req.Port,
		Remark:           req.Remark,
		GeneratedFrom:    "argosbx",
		ChainProxyNodeID: req.ChainProxyNodeID,
	})
	encryptedConfig, err := h.encryptInstallConfig(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt install params failed"})
		return
	}

	var node domain.ProtocolNode
	var task domain.Task
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		node = domain.ProtocolNode{
			UserID:                 userID,
			ServerID:               &server.ID,
			Name:                   req.Name,
			Protocol:               req.Protocol,
			ListenPort:             req.Port,
			PublicPort:             req.PublicPort,
			SubscriptionConfigJSON: string(configJSON),
			EncryptedProtocolJSON:  encryptedConfig,
			InstallMethod:          domain.NodeInstallMethodSystem,
			Status:                 domain.NodeStatusInstalling,
		}
		if err := tx.Create(&node).Error; err != nil {
			return err
		}
		task = domain.Task{
			UserID:   userID,
			ServerID: &server.ID,
			NodeID:   &node.ID,
			Type:     domain.TaskTypeInstall,
			Status:   domain.TaskStatusQueued,
		}
		return tx.Create(&task).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create install task failed"})
		return
	}

	go h.runInstallTask(task.ID, node.ID, server.ID, req)

	h.logOperation(&userID, "node.install.start", "node", map[string]any{"nodeId": node.ID, "taskId": task.ID, "serverId": server.ID})
	c.JSON(http.StatusAccepted, installNodeResponse{
		Node: toNodeResponse(node),
		Task: toTaskResponse(task),
	})
}

func (h *Handler) UninstallNode(c *gin.Context) {
	node, ok := h.findOwnedNode(c)
	if !ok {
		return
	}
	var req uninstallNodeRequest
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&req)
	}
	if node.InstallMethod != domain.NodeInstallMethodSystem {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only system installed nodes can be uninstalled"})
		return
	}
	if node.ServerID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node server is missing"})
		return
	}

	server, ok := h.findServerByID(c, node.UserID, *node.ServerID)
	if !ok {
		return
	}
	if server.Status != domain.ServerStatusNormal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server is not ready for uninstallation"})
		return
	}

	node.Status = domain.NodeStatusUninstalling
	var task domain.Task
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&node).Error; err != nil {
			return err
		}
		task = domain.Task{
			UserID:   node.UserID,
			ServerID: node.ServerID,
			NodeID:   &node.ID,
			Type:     domain.TaskTypeUninstall,
			Status:   domain.TaskStatusQueued,
		}
		return tx.Create(&task).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create uninstall task failed"})
		return
	}

	go h.runUninstallTask(task.ID, node.ID, *node.ServerID, req.DeleteAfterUninstall)

	h.logOperation(&node.UserID, "node.uninstall.start", "node", map[string]any{"nodeId": node.ID, "taskId": task.ID, "serverId": *node.ServerID, "deleteAfterUninstall": req.DeleteAfterUninstall})
	c.JSON(http.StatusAccepted, installNodeResponse{
		Node: toNodeResponse(node),
		Task: toTaskResponse(task),
	})
}

func (h *Handler) runInstallTask(taskID uint, nodeID uint, serverID uint, req installNodeRequest) {
	defer func() {
		if recovered := recover(); recovered != nil {
			h.failInstallTask(taskID, nodeID, fmt.Errorf("install task panic: %v", recovered))
		}
	}()

	h.markTaskRunning(taskID)
	h.appendTaskLog(taskID, "info", "install task started")

	server, err := h.loadServerForTask(serverID)
	if err != nil {
		h.failInstallTask(taskID, nodeID, err)
		return
	}

	targets, err := h.installTargetsForServer(serverID, nodeID, req)
	if err != nil {
		h.failInstallTask(taskID, nodeID, err)
		return
	}
	command, err := argosbx.BuildInstallCommandSet(installParamsForTargets(targets))
	if err != nil {
		h.failInstallTask(taskID, nodeID, err)
		return
	}
	h.appendTaskLog(taskID, "info", fmt.Sprintf("install will apply %d protocol node(s) on this server", len(targets)))
	h.appendTaskLog(taskID, "info", "generated install command: "+maskInstallCommandSet(command, targetRequests(targets)))

	output, err := h.runServerCommand(server, command, func(message string) {
		h.appendTaskLog(taskID, "info", maskInstallCommandSet(message, targetRequests(targets)))
	})
	if err != nil {
		h.failInstallTask(taskID, nodeID, err)
		return
	}
	if err := argosbx.DetectInstallFailure(output); err != nil {
		h.failInstallTask(taskID, nodeID, err)
		return
	}

	now := time.Now()
	h.db.Model(&domain.Task{}).Where("id = ?", taskID).Updates(map[string]any{
		"status":   domain.TaskStatusSuccess,
		"ended_at": now,
	})
	h.updateInstalledTargetsFromOutput(taskID, targets, output)
	h.appendTaskLog(taskID, "info", "install task completed")
}

func (h *Handler) installTargetsForServer(serverID uint, currentNodeID uint, currentReq installNodeRequest) ([]installTarget, error) {
	var nodes []domain.ProtocolNode
	if err := h.db.
		Where("server_id = ? AND install_method = ? AND status = ? AND id <> ?", serverID, domain.NodeInstallMethodSystem, domain.NodeStatusInstallOK, currentNodeID).
		Order("created_at ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	targets := make([]installTarget, 0, len(nodes)+1)
	for _, node := range nodes {
		req, err := h.installRequestFromNode(node)
		if err != nil {
			return nil, fmt.Errorf("prepare existing node %s failed: %w", node.Name, err)
		}
		targets = append(targets, installTarget{NodeID: node.ID, Req: req})
	}
	targets = append(targets, installTarget{NodeID: currentNodeID, Req: currentReq})
	return targets, nil
}

func (h *Handler) installRequestFromNode(node domain.ProtocolNode) (installNodeRequest, error) {
	req := installNodeRequest{
		Name:       node.Name,
		Protocol:   node.Protocol,
		Port:       node.ListenPort,
		PublicPort: node.PublicPort,
		NamePrefix: node.Name,
	}
	if node.ServerID != nil {
		req.ServerID = *node.ServerID
	}
	if strings.TrimSpace(node.EncryptedProtocolJSON) == "" {
		if protocolNeedsUUID(node.Protocol) {
			return installNodeRequest{}, errors.New("encrypted install params are missing")
		}
		return req, nil
	}
	encryptor, err := security.NewEncryptor(h.encryptionKey)
	if err != nil {
		return installNodeRequest{}, err
	}
	plain, err := encryptor.Decrypt(node.EncryptedProtocolJSON)
	if err != nil {
		return installNodeRequest{}, err
	}
	config := encryptedNodeConfig{}
	if err := json.Unmarshal([]byte(plain), &config); err != nil {
		return installNodeRequest{}, err
	}
	values := map[string]string{}
	if strings.TrimSpace(config.Sensitive) != "" {
		_ = json.Unmarshal([]byte(config.Sensitive), &values)
	}
	applyInstallSensitiveValues(&req, values)
	applyInstallRawLinkValues(&req, config.RawLink)
	if protocolNeedsUUID(req.Protocol) && req.UUID == "" {
		return installNodeRequest{}, errors.New("uuid/password is missing")
	}
	return req, nil
}

func applyInstallSensitiveValues(req *installNodeRequest, values map[string]string) {
	if req.UUID == "" {
		req.UUID = strings.TrimSpace(installFirstNonEmpty(values["uuid"], values["password"]))
	}
	req.RealityDomain = installFirstNonEmpty(req.RealityDomain, values["realityDomain"], values["sni"], values["servername"])
	req.CDNDomain = installFirstNonEmpty(req.CDNDomain, values["cdnDomain"])
	req.ArgoMode = installFirstNonEmpty(req.ArgoMode, values["argoMode"])
	req.ArgoDomain = installFirstNonEmpty(req.ArgoDomain, values["argoDomain"])
	req.ArgoToken = installFirstNonEmpty(req.ArgoToken, values["argoToken"])
	req.NamePrefix = installFirstNonEmpty(req.NamePrefix, values["namePrefix"])
}

func applyInstallRawLinkValues(req *installNodeRequest, rawLink string) {
	rawLink = strings.TrimSpace(rawLink)
	if rawLink == "" || strings.HasPrefix(rawLink, "vmess://") {
		return
	}
	values := parseShareLinkValues(rawLink)
	if protocolNeedsUUID(req.Protocol) {
		req.UUID = installFirstNonEmpty(mapValue(values, "uuid", "id", "password", "passwd", "pass"), req.UUID)
	}
	parsed, err := url.Parse(rawLink)
	if err != nil {
		return
	}
	if parsed.User != nil {
		req.UUID = installFirstNonEmpty(parsed.User.Username(), req.UUID)
	}
	if port := parsed.Port(); req.Port == 0 && port != "" {
		if parsedPort, err := strconv.Atoi(port); err == nil {
			req.Port = parsedPort
		}
	}
	query := parsed.Query()
	req.RealityDomain = installFirstNonEmpty(req.RealityDomain, query.Get("sni"), query.Get("servername"))
	req.CDNDomain = installFirstNonEmpty(req.CDNDomain, query.Get("host"))
}

type shareLinkEndpoint struct {
	Address string
	Port    int
}

func shareLinkEndpointFromRawLink(rawLink string) (shareLinkEndpoint, bool) {
	rawLink = strings.TrimSpace(rawLink)
	if rawLink == "" {
		return shareLinkEndpoint{}, false
	}
	values := parseShareLinkValues(rawLink)
	if address := installFirstNonEmpty(mapValue(values, "address", "server", "add")); address != "" {
		if port, err := strconv.Atoi(mapValue(values, "port")); err == nil && port > 0 {
			return shareLinkEndpoint{Address: address, Port: port}, true
		}
	}
	if strings.HasPrefix(strings.ToLower(rawLink), "ss://") {
		return shadowsocksEndpointFromRawLink(rawLink)
	}
	parsed, err := url.Parse(rawLink)
	if err != nil {
		return shareLinkEndpoint{}, false
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil || port == 0 || strings.TrimSpace(parsed.Hostname()) == "" {
		return shareLinkEndpoint{}, false
	}
	return shareLinkEndpoint{Address: parsed.Hostname(), Port: port}, true
}

func shadowsocksEndpointFromRawLink(rawLink string) (shareLinkEndpoint, bool) {
	withoutScheme := strings.TrimPrefix(rawLink, "ss://")
	mainPart := withoutScheme
	if index := strings.IndexAny(mainPart, "?#"); index >= 0 {
		mainPart = mainPart[:index]
	}
	if !strings.Contains(mainPart, "@") {
		decoded := decodeBase64String(mainPart)
		if decoded == "" {
			return shareLinkEndpoint{}, false
		}
		mainPart = decoded
	}
	_, hostPort, ok := strings.Cut(mainPart, "@")
	if !ok {
		return shareLinkEndpoint{}, false
	}
	host, portText, err := net.SplitHostPort(hostPort)
	if err != nil {
		if strings.Count(hostPort, ":") != 1 {
			return shareLinkEndpoint{}, false
		}
		host, portText, _ = strings.Cut(hostPort, ":")
	}
	port, err := strconv.Atoi(strings.TrimSpace(portText))
	if err != nil || port == 0 || strings.TrimSpace(host) == "" {
		return shareLinkEndpoint{}, false
	}
	return shareLinkEndpoint{Address: strings.Trim(strings.TrimSpace(host), "[]"), Port: port}, true
}

func (h *Handler) updatedSubscriptionConfigFromOutput(nodeID uint, endpoint shareLinkEndpoint, rawLink string) (string, bool) {
	var node domain.ProtocolNode
	if err := h.db.Select("subscription_config_json").First(&node, nodeID).Error; err != nil {
		return "", false
	}
	config := nodeConfig{}
	_ = json.Unmarshal([]byte(node.SubscriptionConfigJSON), &config)
	if strings.TrimSpace(config.Address) == "" {
		config.Address = endpoint.Address
	}
	config.Port = endpoint.Port
	config.RawLink = strings.TrimSpace(rawLink)
	return updateNodeConfigJSON(node.SubscriptionConfigJSON, config), true
}

func installFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func installParamsForTargets(targets []installTarget) []argosbx.InstallParams {
	params := make([]argosbx.InstallParams, 0, len(targets))
	for _, target := range targets {
		params = append(params, argosbx.InstallParams{
			Protocol:      target.Req.Protocol,
			Port:          target.Req.Port,
			UUID:          target.Req.UUID,
			RealityDomain: target.Req.RealityDomain,
			CDNDomain:     target.Req.CDNDomain,
			ArgoMode:      target.Req.ArgoMode,
			ArgoDomain:    target.Req.ArgoDomain,
			ArgoToken:     target.Req.ArgoToken,
			NamePrefix:    target.Req.NamePrefix,
		})
	}
	return params
}

func targetRequests(targets []installTarget) []installNodeRequest {
	requests := make([]installNodeRequest, 0, len(targets))
	for _, target := range targets {
		requests = append(requests, target.Req)
	}
	return requests
}

func (h *Handler) updateInstalledTargetsFromOutput(taskID uint, targets []installTarget, output string) {
	rawLinksByProtocol := map[string][]string{}
	for _, target := range targets {
		if _, ok := rawLinksByProtocol[target.Req.Protocol]; !ok {
			rawLinksByProtocol[target.Req.Protocol] = argosbx.ExtractShareLinks(output, target.Req.Protocol)
		}
		rawLinks := rawLinksByProtocol[target.Req.Protocol]
		rawLink := ""
		if len(rawLinks) > 0 {
			rawLink = rawLinks[0]
			rawLinksByProtocol[target.Req.Protocol] = rawLinks[1:]
		}
		nodeUpdates := map[string]any{"status": domain.NodeStatusInstallOK}
		if rawLink != "" {
			req := target.Req
			applyInstallRawLinkValues(&req, rawLink)
			if encryptedConfig, err := h.encryptInstallConfigWithRawLink(req, rawLink); err == nil {
				nodeUpdates["encrypted_protocol_json"] = encryptedConfig
			} else {
				h.appendTaskLog(taskID, "error", "encrypt extracted share link failed: "+err.Error())
			}
			if endpoint, ok := shareLinkEndpointFromRawLink(rawLink); ok {
				nodeUpdates["listen_port"] = endpoint.Port
				if configJSON, ok := h.updatedSubscriptionConfigFromOutput(target.NodeID, endpoint, rawLink); ok {
					nodeUpdates["subscription_config_json"] = configJSON
				}
			}
		}
		h.db.Model(&domain.ProtocolNode{}).Where("id = ?", target.NodeID).Updates(nodeUpdates)
	}
}

func prepareInstallRequest(req installNodeRequest) (installNodeRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Protocol = strings.TrimSpace(req.Protocol)
	req.UUID = strings.TrimSpace(req.UUID)
	req.RealityDomain = strings.TrimSpace(req.RealityDomain)
	req.CDNDomain = strings.TrimSpace(req.CDNDomain)
	req.ArgoMode = strings.TrimSpace(req.ArgoMode)
	req.ArgoDomain = strings.TrimSpace(req.ArgoDomain)
	req.ArgoToken = strings.TrimSpace(req.ArgoToken)
	req.NamePrefix = strings.TrimSpace(req.NamePrefix)
	req.Remark = strings.TrimSpace(req.Remark)

	var err error
	if req.Port == 0 {
		req.Port, err = randomPort()
		if err != nil {
			return installNodeRequest{}, err
		}
	}
	if req.PublicPort != nil && (*req.PublicPort < 1 || *req.PublicPort > 65535) {
		return installNodeRequest{}, errors.New("node public port must be between 1 and 65535")
	}
	if err := validateInstallRequiredFields(req); err != nil {
		return installNodeRequest{}, err
	}
	if req.UUID == "" && protocolNeedsUUID(req.Protocol) {
		if strings.Contains(strings.ToLower(req.Protocol), "shadowsocks") {
			req.UUID, err = randomSS2022Key()
		} else {
			req.UUID, err = randomUUID()
		}
		if err != nil {
			return installNodeRequest{}, err
		}
	}
	if req.NamePrefix == "" {
		req.NamePrefix = req.Name
	}
	return req, nil
}

func validateInstallRequiredFields(req installNodeRequest) error {
	protocol := strings.ReplaceAll(strings.ToLower(req.Protocol), " ", "")
	if protocol == "argo固定隧道" {
		if req.ArgoDomain == "" {
			return errors.New("argo domain is required for fixed argo tunnel")
		}
		if req.ArgoToken == "" {
			return errors.New("argo token is required for fixed argo tunnel")
		}
	}
	return nil
}

func (h *Handler) encryptInstallConfig(req installNodeRequest) (string, error) {
	return h.encryptInstallConfigWithRawLink(req, "")
}

func (h *Handler) encryptInstallConfigWithRawLink(req installNodeRequest, rawLink string) (string, error) {
	sensitive := map[string]string{}
	if rawLink != "" {
		for key, value := range parseShareLinkValues(rawLink) {
			if strings.TrimSpace(value) != "" {
				sensitive[key] = strings.TrimSpace(value)
			}
		}
	}
	if req.UUID != "" && sensitive["uuid"] == "" {
		sensitive["uuid"] = req.UUID
	}
	if strings.Contains(strings.ToLower(req.Protocol), "shadowsocks") && req.UUID != "" && sensitive["password"] == "" {
		sensitive["password"] = req.UUID
		sensitive["cipher"] = "2022-blake3-aes-128-gcm"
	}
	if req.ArgoToken != "" {
		sensitive["argoToken"] = req.ArgoToken
	}
	if req.RealityDomain != "" {
		sensitive["realityDomain"] = req.RealityDomain
	}
	if req.CDNDomain != "" {
		sensitive["cdnDomain"] = req.CDNDomain
	}
	if req.ArgoMode != "" {
		sensitive["argoMode"] = req.ArgoMode
	}
	if req.ArgoDomain != "" {
		sensitive["argoDomain"] = req.ArgoDomain
	}
	if req.NamePrefix != "" {
		sensitive["namePrefix"] = req.NamePrefix
	}
	if len(sensitive) == 0 {
		if rawLink == "" {
			return "", nil
		}
		return h.encryptNodeConfig(encryptedNodeConfig{RawLink: rawLink})
	}
	data, err := json.Marshal(sensitive)
	if err != nil {
		return "", err
	}
	return h.encryptNodeConfig(encryptedNodeConfig{
		Sensitive: string(data),
		RawLink:   strings.TrimSpace(rawLink),
	})
}

func protocolNeedsUUID(protocol string) bool {
	value := strings.ToLower(protocol)
	return strings.Contains(value, "vless") ||
		strings.Contains(value, "vmess") ||
		strings.Contains(value, "reality") ||
		strings.Contains(value, "anytls") ||
		strings.Contains(value, "shadowsocks") ||
		strings.Contains(value, "argo")
}

func randomPort() (int, error) {
	bytes := make([]byte, 2)
	if _, err := rand.Read(bytes); err != nil {
		return 0, err
	}
	value := int(bytes[0])<<8 + int(bytes[1])
	return 20_000 + value%30_000, nil
}

func randomUUID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16]), nil
}

func randomSS2022Key() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

func maskInstallCommand(command string, req installNodeRequest) string {
	return maskInstallCommandSet(command, []installNodeRequest{req})
}

func maskInstallCommandSet(command string, requests []installNodeRequest) string {
	replacements := []string{}
	for _, req := range requests {
		replacements = append(replacements, req.UUID, req.ArgoToken)
	}
	return maskSensitiveValues(command, replacements)
}

func maskSensitiveValues(command string, replacements []string) string {
	masked := command
	for _, value := range replacements {
		if value == "" {
			continue
		}
		masked = strings.ReplaceAll(masked, value, "***")
	}
	return masked
}

func (h *Handler) runUninstallTask(taskID uint, nodeID uint, serverID uint, deleteAfterUninstall bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			h.failUninstallTask(taskID, nodeID, fmt.Errorf("uninstall task panic: %v", recovered))
		}
	}()

	h.markTaskRunning(taskID)
	h.appendTaskLog(taskID, "info", "uninstall task started")

	server, err := h.loadServerForTask(serverID)
	if err != nil {
		h.failUninstallTask(taskID, nodeID, err)
		return
	}

	remainingTargets, err := h.remainingInstallTargetsForServer(serverID, nodeID)
	if err != nil {
		h.failUninstallTask(taskID, nodeID, err)
		return
	}
	if len(remainingTargets) == 0 {
		command := argosbx.BuildUninstallCommand()
		h.appendTaskLog(taskID, "info", "generated uninstall command: "+command)
		_, err = h.runServerCommand(server, command, func(message string) {
			h.appendTaskLog(taskID, "info", message)
		})
	} else {
		command, buildErr := argosbx.BuildInstallCommandSet(installParamsForTargets(remainingTargets))
		if buildErr != nil {
			h.failUninstallTask(taskID, nodeID, buildErr)
			return
		}
		h.appendTaskLog(taskID, "info", fmt.Sprintf("uninstall will re-apply %d remaining protocol node(s) on this server", len(remainingTargets)))
		h.appendTaskLog(taskID, "info", "generated remaining-node install command: "+maskInstallCommandSet(command, targetRequests(remainingTargets)))
		var output string
		output, err = h.runServerCommand(server, command, func(message string) {
			h.appendTaskLog(taskID, "info", maskInstallCommandSet(message, targetRequests(remainingTargets)))
		})
		if err == nil {
			if detectErr := argosbx.DetectInstallFailure(output); detectErr != nil {
				err = detectErr
			} else {
				h.updateInstalledTargetsFromOutput(taskID, remainingTargets, output)
			}
		}
	}
	if err != nil {
		h.failUninstallTask(taskID, nodeID, err)
		return
	}

	now := time.Now()
	h.db.Model(&domain.Task{}).Where("id = ?", taskID).Updates(map[string]any{
		"status":   domain.TaskStatusSuccess,
		"ended_at": now,
	})
	if deleteAfterUninstall {
		if err := h.deleteNodeRecord(nodeID); err != nil {
			h.failUninstallTask(taskID, nodeID, err)
			return
		}
		h.appendTaskLog(taskID, "info", "node record deleted after uninstall")
	} else {
		h.db.Model(&domain.ProtocolNode{}).Where("id = ?", nodeID).Update("status", domain.NodeStatusUninstalled)
	}
	h.appendTaskLog(taskID, "info", "uninstall task completed")
}

func (h *Handler) remainingInstallTargetsForServer(serverID uint, uninstalledNodeID uint) ([]installTarget, error) {
	var nodes []domain.ProtocolNode
	if err := h.db.
		Where("server_id = ? AND install_method = ? AND status = ? AND id <> ?", serverID, domain.NodeInstallMethodSystem, domain.NodeStatusInstallOK, uninstalledNodeID).
		Order("created_at ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	targets := make([]installTarget, 0, len(nodes))
	for _, node := range nodes {
		req, err := h.installRequestFromNode(node)
		if err != nil {
			return nil, fmt.Errorf("prepare remaining node %s failed: %w", node.Name, err)
		}
		targets = append(targets, installTarget{NodeID: node.ID, Req: req})
	}
	return targets, nil
}

func (h *Handler) markTaskRunning(taskID uint) {
	now := time.Now()
	h.db.Model(&domain.Task{}).Where("id = ?", taskID).Updates(map[string]any{
		"status":     domain.TaskStatusRunning,
		"started_at": now,
	})
}

func (h *Handler) failInstallTask(taskID uint, nodeID uint, err error) {
	now := time.Now()
	h.db.Model(&domain.Task{}).Where("id = ?", taskID).Updates(map[string]any{
		"status":   domain.TaskStatusFailed,
		"error":    err.Error(),
		"ended_at": now,
	})
	h.db.Model(&domain.ProtocolNode{}).Where("id = ?", nodeID).Update("status", domain.NodeStatusInstallFailed)
	h.appendTaskLog(taskID, "error", err.Error())
}

func (h *Handler) failUninstallTask(taskID uint, nodeID uint, err error) {
	now := time.Now()
	h.db.Model(&domain.Task{}).Where("id = ?", taskID).Updates(map[string]any{
		"status":   domain.TaskStatusFailed,
		"error":    err.Error(),
		"ended_at": now,
	})
	h.db.Model(&domain.ProtocolNode{}).Where("id = ?", nodeID).Update("status", domain.NodeStatusInstallOK)
	h.appendTaskLog(taskID, "error", err.Error())
}

func (h *Handler) appendTaskLog(taskID uint, level string, message string) {
	h.db.Create(&domain.TaskLog{
		TaskID:  taskID,
		Level:   level,
		Message: message,
	})
}

func (h *Handler) loadServerForTask(serverID uint) (domain.Server, error) {
	var server domain.Server
	if err := h.db.First(&server, serverID).Error; err != nil {
		return domain.Server{}, err
	}
	return server, nil
}

func (h *Handler) runServerCommand(server domain.Server, command string, onOutput func(string)) (string, error) {
	encryptor, err := security.NewEncryptor(h.encryptionKey)
	if err != nil {
		return "", err
	}
	req, err := h.serverToTestRequest(encryptor, server)
	if err != nil {
		return "", err
	}
	return sshclient.RunCommandWithOutput(context.Background(), sshclient.TestRequest{
		Host:       req.Host,
		Port:       req.SSHPort,
		Username:   req.SSHUsername,
		AuthMethod: sshclient.AuthMethod(req.AuthMethod),
		Password:   req.Password,
		PrivateKey: req.PrivateKey,
		Timeout:    30 * time.Minute,
	}, command, onOutput)
}

func (h *Handler) findServerByID(c *gin.Context, userID uint, serverID uint) (domain.Server, bool) {
	var server domain.Server
	if err := h.db.Where("id = ? AND user_id = ?", serverID, userID).First(&server).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return domain.Server{}, false
	}
	return server, true
}
