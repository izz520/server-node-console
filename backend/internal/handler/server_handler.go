package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"server-sing-box-2/backend/internal/domain"
	"server-sing-box-2/backend/internal/middleware"
	"server-sing-box-2/backend/internal/security"
	"server-sing-box-2/backend/internal/sshclient"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type serverRequest struct {
	Name         string            `json:"name" binding:"required,max=120"`
	Host         string            `json:"host" binding:"required,max=255"`
	SSHPort      int               `json:"sshPort" binding:"required,min=1,max=65535"`
	SSHUsername  string            `json:"sshUsername" binding:"required,max=120"`
	AuthMethod   domain.AuthMethod `json:"authMethod" binding:"required"`
	Password     string            `json:"password"`
	PrivateKey   string            `json:"privateKey"`
	Region       string            `json:"region"`
	Tags         string            `json:"tags"`
	Remark       string            `json:"remark"`
	ExpiresAt    *time.Time        `json:"expiresAt"`
	Price        float64           `json:"price"`
	BillingCycle string            `json:"billingCycle"`
	Currency     string            `json:"currency"`
}

type serverResponse struct {
	ID            uint                `json:"id"`
	UserID        uint                `json:"userId"`
	Name          string              `json:"name"`
	Host          string              `json:"host"`
	SSHPort       int                 `json:"sshPort"`
	SSHUsername   string              `json:"sshUsername"`
	AuthMethod    domain.AuthMethod   `json:"authMethod"`
	Region        string              `json:"region"`
	Tags          string              `json:"tags"`
	Remark        string              `json:"remark"`
	Status        domain.ServerStatus `json:"status"`
	HasPassword   bool                `json:"hasPassword"`
	HasPrivateKey bool                `json:"hasPrivateKey"`
	LastCheckedAt *time.Time          `json:"lastCheckedAt"`
	CreatedAt     time.Time           `json:"createdAt"`
	UpdatedAt     time.Time           `json:"updatedAt"`
	ExpiresAt     *time.Time          `json:"expiresAt"`
	Price         float64             `json:"price"`
	BillingCycle  string              `json:"billingCycle"`
	Currency      string              `json:"currency"`
}

func (h *Handler) ListServers(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var servers []domain.Server
	if err := h.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list servers failed"})
		return
	}

	items := make([]serverResponse, 0, len(servers))
	for _, server := range servers {
		items = append(items, toServerResponse(server))
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) GetServer(c *gin.Context) {
	server, ok := h.findOwnedServer(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, toServerResponse(server))
}

func (h *Handler) CreateServer(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req serverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	req = normalizeServerRequest(req)
	if err := validateServerCredential(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := testSSH(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ssh connection failed", "details": err.Error()})
		return
	}

	encryptor, err := security.NewEncryptor(h.encryptionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption is not configured"})
		return
	}

	encryptedPassword, encryptedPrivateKey, err := encryptServerCredentials(encryptor, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt ssh credential failed"})
		return
	}

	now := time.Now()
	server := domain.Server{
		UserID:              userID,
		Name:                req.Name,
		Host:                req.Host,
		SSHPort:             req.SSHPort,
		SSHUsername:         req.SSHUsername,
		AuthMethod:          req.AuthMethod,
		EncryptedPassword:   encryptedPassword,
		EncryptedPrivateKey: encryptedPrivateKey,
		Region:              req.Region,
		Tags:                req.Tags,
		Remark:              req.Remark,
		Status:              domain.ServerStatusNormal,
		LastCheckedAt:       &now,
		ExpiresAt:           req.ExpiresAt,
		Price:               req.Price,
		BillingCycle:        req.BillingCycle,
		Currency:            req.Currency,
	}
	if err := h.db.Create(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create server failed"})
		return
	}

	h.logOperation(&userID, "server.create", "server", map[string]any{"serverId": server.ID, "name": server.Name})
	c.JSON(http.StatusCreated, toServerResponse(server))
}

func (h *Handler) UpdateServer(c *gin.Context) {
	server, ok := h.findOwnedServer(c)
	if !ok {
		return
	}

	var req serverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	req = normalizeServerRequest(req)
	encryptor, err := security.NewEncryptor(h.encryptionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption is not configured"})
		return
	}

	effectiveReq, err := h.effectiveServerRequest(encryptor, server, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateServerCredential(effectiveReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if serverConnectionChanged(server, req) || req.Password != "" || req.PrivateKey != "" {
		if err := testSSH(c.Request.Context(), effectiveReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ssh connection failed", "details": err.Error()})
			return
		}
		now := time.Now()
		server.LastCheckedAt = &now
		server.Status = domain.ServerStatusNormal
	}

	encryptedPassword, encryptedPrivateKey, err := encryptServerCredentials(encryptor, effectiveReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt ssh credential failed"})
		return
	}

	server.Name = req.Name
	server.Host = req.Host
	server.SSHPort = req.SSHPort
	server.SSHUsername = req.SSHUsername
	server.AuthMethod = req.AuthMethod
	server.EncryptedPassword = encryptedPassword
	server.EncryptedPrivateKey = encryptedPrivateKey
	server.Region = req.Region
	server.Tags = req.Tags
	server.Remark = req.Remark
	server.ExpiresAt = req.ExpiresAt
	server.Price = req.Price
	server.BillingCycle = req.BillingCycle
	server.Currency = req.Currency

	if err := h.db.Save(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update server failed"})
		return
	}

	h.logOperation(&server.UserID, "server.update", "server", map[string]any{"serverId": server.ID, "name": server.Name})
	c.JSON(http.StatusOK, toServerResponse(server))
}

func (h *Handler) DeleteServer(c *gin.Context) {
	server, ok := h.findOwnedServer(c)
	if !ok {
		return
	}

	if blocked, message := h.serverDeletionBlocked(server); blocked {
		c.JSON(http.StatusConflict, gin.H{"error": message})
		return
	}

	if err := h.db.Delete(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete server failed"})
		return
	}

	h.logOperation(&server.UserID, "server.delete", "server", map[string]any{"serverId": server.ID, "name": server.Name})
	c.Status(http.StatusNoContent)
}

func (h *Handler) serverDeletionBlocked(server domain.Server) (bool, string) {
	var natMappingCount int64
	if err := h.db.Model(&domain.NATPortMapping{}).
		Where("user_id = ? AND server_id = ?", server.UserID, server.ID).
		Count(&natMappingCount).Error; err != nil {
		return true, "check server NAT mappings failed"
	}
	if natMappingCount > 0 {
		return true, "server has NAT mappings; delete NAT mappings first"
	}

	var subscriptionRefCount int64
	if err := h.db.Model(&domain.SubscriptionNode{}).
		Joins("JOIN protocol_nodes ON protocol_nodes.id = subscription_nodes.node_id").
		Where("protocol_nodes.user_id = ? AND protocol_nodes.server_id = ? AND protocol_nodes.deleted_at IS NULL", server.UserID, server.ID).
		Count(&subscriptionRefCount).Error; err != nil {
		return true, "check server references failed"
	}
	if subscriptionRefCount > 0 {
		return true, "server has nodes referenced by subscriptions; remove them from subscriptions first"
	}

	var nodeCount int64
	if err := h.db.Model(&domain.ProtocolNode{}).
		Where("user_id = ? AND server_id = ?", server.UserID, server.ID).
		Count(&nodeCount).Error; err != nil {
		return true, "check server nodes failed"
	}
	if nodeCount > 0 {
		return true, "server has nodes; uninstall or delete nodes first"
	}

	return false, ""
}

func (h *Handler) TestServerSSH(c *gin.Context) {
	server, ok := h.findOwnedServer(c)
	if !ok {
		return
	}

	encryptor, err := security.NewEncryptor(h.encryptionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption is not configured"})
		return
	}

	req, err := h.serverToTestRequest(encryptor, server)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	if err := testSSH(c.Request.Context(), req); err != nil {
		server.Status = domain.ServerStatusConnectionFailed
		server.LastCheckedAt = &now
		_ = h.db.Save(&server).Error
		h.logOperation(&server.UserID, "server.test_ssh.failed", "server", map[string]any{"serverId": server.ID, "name": server.Name})
		c.JSON(http.StatusBadRequest, gin.H{"error": "ssh connection failed", "details": err.Error(), "server": toServerResponse(server)})
		return
	}

	server.Status = domain.ServerStatusNormal
	server.LastCheckedAt = &now
	if err := h.db.Save(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update server status failed"})
		return
	}

	h.logOperation(&server.UserID, "server.test_ssh.success", "server", map[string]any{"serverId": server.ID, "name": server.Name})
	c.JSON(http.StatusOK, gin.H{"status": "ok", "server": toServerResponse(server)})
}

func (h *Handler) findOwnedServer(c *gin.Context) (domain.Server, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		return domain.Server{}, false
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid server id"})
		return domain.Server{}, false
	}

	var server domain.Server
	err = h.db.Where("id = ? AND user_id = ?", id, userID).First(&server).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return domain.Server{}, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get server failed"})
		return domain.Server{}, false
	}

	return server, true
}

func (h *Handler) effectiveServerRequest(encryptor *security.Encryptor, server domain.Server, req serverRequest) (serverRequest, error) {
	effectiveReq := req
	if req.AuthMethod == domain.AuthMethodPassword && req.Password == "" {
		password, err := encryptor.Decrypt(server.EncryptedPassword)
		if err != nil {
			return serverRequest{}, err
		}
		effectiveReq.Password = password
	}
	if req.AuthMethod == domain.AuthMethodPrivateKey && req.PrivateKey == "" {
		privateKey, err := encryptor.Decrypt(server.EncryptedPrivateKey)
		if err != nil {
			return serverRequest{}, err
		}
		effectiveReq.PrivateKey = privateKey
	}
	return effectiveReq, nil
}

func (h *Handler) serverToTestRequest(encryptor *security.Encryptor, server domain.Server) (serverRequest, error) {
	req := serverRequest{
		Name:        server.Name,
		Host:        server.Host,
		SSHPort:     server.SSHPort,
		SSHUsername: server.SSHUsername,
		AuthMethod:  server.AuthMethod,
	}

	switch server.AuthMethod {
	case domain.AuthMethodPassword:
		password, err := encryptor.Decrypt(server.EncryptedPassword)
		if err != nil {
			return serverRequest{}, err
		}
		req.Password = password
	case domain.AuthMethodPrivateKey:
		privateKey, err := encryptor.Decrypt(server.EncryptedPrivateKey)
		if err != nil {
			return serverRequest{}, err
		}
		req.PrivateKey = privateKey
	}

	return req, nil
}

func currentUserID(c *gin.Context) (uint, bool) {
	value, ok := c.Get(middleware.ContextUserID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
		return 0, false
	}

	userID, ok := value.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return 0, false
	}
	return userID, true
}

func normalizeServerRequest(req serverRequest) serverRequest {
	req.Name = strings.TrimSpace(req.Name)
	req.Host = strings.TrimSpace(req.Host)
	req.SSHUsername = strings.TrimSpace(req.SSHUsername)
	req.Region = strings.TrimSpace(req.Region)
	req.Tags = strings.TrimSpace(req.Tags)
	req.Remark = strings.TrimSpace(req.Remark)
	req.BillingCycle = strings.TrimSpace(req.BillingCycle)
	req.Currency = strings.TrimSpace(req.Currency)
	return req
}

func validateServerCredential(req serverRequest) error {
	switch req.AuthMethod {
	case domain.AuthMethodPassword:
		if req.Password == "" {
			return errors.New("ssh password is required")
		}
	case domain.AuthMethodPrivateKey:
		if req.PrivateKey == "" {
			return errors.New("ssh private key is required")
		}
	default:
		return errors.New("unsupported ssh auth method")
	}
	return nil
}

func testSSH(ctx context.Context, req serverRequest) error {
	return sshclient.TestConnection(ctx, sshclient.TestRequest{
		Host:       req.Host,
		Port:       req.SSHPort,
		Username:   req.SSHUsername,
		AuthMethod: sshclient.AuthMethod(req.AuthMethod),
		Password:   req.Password,
		PrivateKey: req.PrivateKey,
		Timeout:    10 * time.Second,
	})
}

func encryptServerCredentials(encryptor *security.Encryptor, req serverRequest) (string, string, error) {
	switch req.AuthMethod {
	case domain.AuthMethodPassword:
		password, err := encryptor.Encrypt(req.Password)
		return password, "", err
	case domain.AuthMethodPrivateKey:
		privateKey, err := encryptor.Encrypt(req.PrivateKey)
		return "", privateKey, err
	default:
		return "", "", errors.New("unsupported ssh auth method")
	}
}

func serverConnectionChanged(server domain.Server, req serverRequest) bool {
	return server.Host != req.Host ||
		server.SSHPort != req.SSHPort ||
		server.SSHUsername != req.SSHUsername ||
		server.AuthMethod != req.AuthMethod
}

func toServerResponse(server domain.Server) serverResponse {
	return serverResponse{
		ID:            server.ID,
		UserID:        server.UserID,
		Name:          server.Name,
		Host:          server.Host,
		SSHPort:       server.SSHPort,
		SSHUsername:   server.SSHUsername,
		AuthMethod:    server.AuthMethod,
		Region:        server.Region,
		Tags:          server.Tags,
		Remark:        server.Remark,
		Status:        server.Status,
		HasPassword:   server.EncryptedPassword != "",
		HasPrivateKey: server.EncryptedPrivateKey != "",
		LastCheckedAt: server.LastCheckedAt,
		CreatedAt:     server.CreatedAt,
		UpdatedAt:     server.UpdatedAt,
		ExpiresAt:     server.ExpiresAt,
		Price:         server.Price,
		BillingCycle:  server.BillingCycle,
		Currency:      server.Currency,
	}
}
