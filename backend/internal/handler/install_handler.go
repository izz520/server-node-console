package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
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
	ServerID      uint   `json:"serverId" binding:"required"`
	Name          string `json:"name" binding:"required,max=120"`
	Protocol      string `json:"protocol" binding:"required,max=120"`
	Port          int    `json:"port" binding:"min=0,max=65535"`
	UUID          string `json:"uuid"`
	RealityDomain string `json:"realityDomain"`
	CDNDomain     string `json:"cdnDomain"`
	ArgoMode      string `json:"argoMode"`
	ArgoDomain    string `json:"argoDomain"`
	ArgoToken     string `json:"argoToken"`
	NamePrefix    string `json:"namePrefix"`
	Remark        string `json:"remark"`
}

type installNodeResponse struct {
	Node nodeResponse `json:"node"`
	Task taskResponse `json:"task"`
}

type installConfig struct {
	Address       string `json:"address"`
	Port          int    `json:"port"`
	Remark        string `json:"remark,omitempty"`
	GeneratedFrom string `json:"generatedFrom"`
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "prepare install params failed"})
		return
	}
	if _, err := argosbx.VarNameForProtocol(req.Protocol); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	configJSON, _ := json.Marshal(installConfig{
		Address:       server.Host,
		Port:          req.Port,
		Remark:        req.Remark,
		GeneratedFrom: "argosbx",
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

	go h.runUninstallTask(task.ID, node.ID, *node.ServerID)

	h.logOperation(&node.UserID, "node.uninstall.start", "node", map[string]any{"nodeId": node.ID, "taskId": task.ID, "serverId": *node.ServerID})
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

	command, _, err := argosbx.BuildInstallCommand(argosbx.InstallParams{
		Protocol:      req.Protocol,
		Port:          req.Port,
		UUID:          req.UUID,
		RealityDomain: req.RealityDomain,
		CDNDomain:     req.CDNDomain,
		ArgoMode:      req.ArgoMode,
		ArgoDomain:    req.ArgoDomain,
		ArgoToken:     req.ArgoToken,
		NamePrefix:    req.NamePrefix,
	})
	if err != nil {
		h.failInstallTask(taskID, nodeID, err)
		return
	}
	h.appendTaskLog(taskID, "info", "generated install command: "+maskInstallCommand(command, req))

	output, err := h.runServerCommand(server, command)
	if output != "" {
		h.appendTaskLog(taskID, "info", output)
	}
	if err != nil {
		h.failInstallTask(taskID, nodeID, err)
		return
	}

	now := time.Now()
	h.db.Model(&domain.Task{}).Where("id = ?", taskID).Updates(map[string]any{
		"status":   domain.TaskStatusSuccess,
		"ended_at": now,
	})
	h.db.Model(&domain.ProtocolNode{}).Where("id = ?", nodeID).Update("status", domain.NodeStatusInstallOK)
	h.appendTaskLog(taskID, "info", "install task completed")
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
	if req.UUID == "" && protocolNeedsUUID(req.Protocol) {
		req.UUID, err = randomUUID()
		if err != nil {
			return installNodeRequest{}, err
		}
	}
	if req.NamePrefix == "" {
		req.NamePrefix = req.Name
	}
	return req, nil
}

func (h *Handler) encryptInstallConfig(req installNodeRequest) (string, error) {
	sensitive := map[string]string{}
	if req.UUID != "" {
		sensitive["uuid"] = req.UUID
	}
	if req.ArgoToken != "" {
		sensitive["argoToken"] = req.ArgoToken
	}
	if len(sensitive) == 0 {
		return "", nil
	}
	data, err := json.Marshal(sensitive)
	if err != nil {
		return "", err
	}
	return h.encryptNodeConfig(encryptedNodeConfig{Sensitive: string(data)})
}

func protocolNeedsUUID(protocol string) bool {
	value := strings.ToLower(protocol)
	return strings.Contains(value, "vless") ||
		strings.Contains(value, "vmess") ||
		strings.Contains(value, "reality") ||
		strings.Contains(value, "anytls") ||
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

func maskInstallCommand(command string, req installNodeRequest) string {
	replacements := []string{req.UUID, req.ArgoToken}
	masked := command
	for _, value := range replacements {
		if value == "" {
			continue
		}
		masked = strings.ReplaceAll(masked, value, "***")
	}
	return masked
}

func (h *Handler) runUninstallTask(taskID uint, nodeID uint, serverID uint) {
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

	command := argosbx.BuildUninstallCommand()
	h.appendTaskLog(taskID, "info", "generated uninstall command: "+command)
	output, err := h.runServerCommand(server, command)
	if output != "" {
		h.appendTaskLog(taskID, "info", output)
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
	h.db.Model(&domain.ProtocolNode{}).Where("id = ?", nodeID).Update("status", domain.NodeStatusUninstalled)
	h.appendTaskLog(taskID, "info", "uninstall task completed")
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

func (h *Handler) runServerCommand(server domain.Server, command string) (string, error) {
	encryptor, err := security.NewEncryptor(h.encryptionKey)
	if err != nil {
		return "", err
	}
	req, err := h.serverToTestRequest(encryptor, server)
	if err != nil {
		return "", err
	}
	return sshclient.RunCommand(context.Background(), sshclient.TestRequest{
		Host:       req.Host,
		Port:       req.SSHPort,
		Username:   req.SSHUsername,
		AuthMethod: sshclient.AuthMethod(req.AuthMethod),
		Password:   req.Password,
		PrivateKey: req.PrivateKey,
		Timeout:    30 * time.Minute,
	}, command)
}

func (h *Handler) findServerByID(c *gin.Context, userID uint, serverID uint) (domain.Server, bool) {
	var server domain.Server
	if err := h.db.Where("id = ? AND user_id = ?", serverID, userID).First(&server).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return domain.Server{}, false
	}
	return server, true
}
