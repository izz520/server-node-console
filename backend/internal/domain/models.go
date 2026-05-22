package domain

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

type ServerStatus string

const (
	ServerStatusNormal           ServerStatus = "normal"
	ServerStatusConnectionFailed ServerStatus = "connection_failed"
	ServerStatusDisabled         ServerStatus = "disabled"
)

type AuthMethod string

const (
	AuthMethodPassword   AuthMethod = "password"
	AuthMethodPrivateKey AuthMethod = "private_key"
)

type NodeStatus string

const (
	NodeStatusInstalling    NodeStatus = "installing"
	NodeStatusInstallOK     NodeStatus = "install_success"
	NodeStatusInstallFailed NodeStatus = "install_failed"
	NodeStatusUninstalling  NodeStatus = "uninstalling"
	NodeStatusUninstalled   NodeStatus = "uninstalled"
	NodeStatusImported      NodeStatus = "imported"
)

type NodeInstallMethod string

const (
	NodeInstallMethodSystem   NodeInstallMethod = "system"
	NodeInstallMethodExternal NodeInstallMethod = "external"
)

type TaskType string

const (
	TaskTypeInstall   TaskType = "install"
	TaskTypeUninstall TaskType = "uninstall"
	TaskTypeSSHTest   TaskType = "ssh_test"
)

type TaskStatus string

const (
	TaskStatusQueued  TaskStatus = "queued"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusSuccess TaskStatus = "success"
	TaskStatusFailed  TaskStatus = "failed"
)

type SubscriptionFormat string

const (
	SubscriptionFormatSingBox      SubscriptionFormat = "sing-box"
	SubscriptionFormatClashMihomo  SubscriptionFormat = "clash-mihomo"
	SubscriptionFormatV2RayN       SubscriptionFormat = "v2rayn"
	SubscriptionFormatShadowrocket SubscriptionFormat = "shadowrocket"
	SubscriptionFormatBase64       SubscriptionFormat = "base64"
)

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Email        string         `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	Role         UserRole       `gorm:"size:32;not null;default:user" json:"role"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type Server struct {
	ID                  uint           `gorm:"primaryKey" json:"id"`
	UserID              uint           `gorm:"index;not null" json:"userId"`
	Name                string         `gorm:"size:120;not null" json:"name"`
	Host                string         `gorm:"size:255;not null" json:"host"`
	SSHPort             int            `gorm:"not null;default:22" json:"sshPort"`
	SSHUsername         string         `gorm:"size:120;not null" json:"sshUsername"`
	AuthMethod          AuthMethod     `gorm:"size:32;not null" json:"authMethod"`
	EncryptedPassword   string         `gorm:"type:text" json:"-"`
	EncryptedPrivateKey string         `gorm:"type:text" json:"-"`
	Region              string         `gorm:"size:120" json:"region"`
	Tags                string         `gorm:"type:text" json:"tags"`
	Remark              string         `gorm:"type:text" json:"remark"`
	Status              ServerStatus   `gorm:"size:32;not null;default:connection_failed" json:"status"`
	LastCheckedAt       *time.Time     `json:"lastCheckedAt"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

type NATPortMapping struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"index;not null" json:"userId"`
	ServerID   uint           `gorm:"index;not null" json:"serverId"`
	Name       string         `gorm:"size:120;not null" json:"name"`
	Transport  string         `gorm:"size:16" json:"transport"`
	ListenPort int            `gorm:"not null" json:"listenPort"`
	PublicPort int            `gorm:"not null" json:"publicPort"`
	Remark     string         `gorm:"type:text" json:"remark"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type ProtocolNode struct {
	ID                     uint              `gorm:"primaryKey" json:"id"`
	UserID                 uint              `gorm:"index;not null" json:"userId"`
	ServerID               *uint             `gorm:"index" json:"serverId"`
	Name                   string            `gorm:"size:120;not null" json:"name"`
	Protocol               string            `gorm:"size:120;not null" json:"protocol"`
	ListenPort             int               `json:"listenPort"`
	PublicPort             *int              `json:"publicPort"`
	EncryptedProtocolJSON  string            `gorm:"type:text" json:"-"`
	SubscriptionConfigJSON string            `gorm:"type:jsonb;default:'{}'" json:"subscriptionConfig"`
	InstallMethod          NodeInstallMethod `gorm:"size:32;not null" json:"installMethod"`
	Status                 NodeStatus        `gorm:"size:32;not null" json:"status"`
	CreatedAt              time.Time         `json:"createdAt"`
	UpdatedAt              time.Time         `json:"updatedAt"`
	DeletedAt              gorm.DeletedAt    `gorm:"index" json:"-"`
}

type Subscription struct {
	ID        uint               `gorm:"primaryKey" json:"id"`
	UserID    uint               `gorm:"index;not null" json:"userId"`
	Name      string             `gorm:"size:120;not null" json:"name"`
	TokenHash string             `gorm:"size:255;uniqueIndex;not null" json:"-"`
	Enabled   bool               `gorm:"not null;default:true" json:"enabled"`
	Format    SubscriptionFormat `gorm:"size:32;not null" json:"format"`
	Remark    string             `gorm:"type:text" json:"remark"`
	CreatedAt time.Time          `json:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt"`
	DeletedAt gorm.DeletedAt     `gorm:"index" json:"-"`
}

type SubscriptionNode struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SubscriptionID uint      `gorm:"uniqueIndex:idx_subscription_node;not null" json:"subscriptionId"`
	NodeID         uint      `gorm:"uniqueIndex:idx_subscription_node;not null" json:"nodeId"`
	SortOrder      int       `gorm:"not null;default:0" json:"sortOrder"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Task struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"userId"`
	ServerID  *uint          `gorm:"index" json:"serverId"`
	NodeID    *uint          `gorm:"index" json:"nodeId"`
	Type      TaskType       `gorm:"size:32;not null" json:"type"`
	Status    TaskStatus     `gorm:"size:32;not null;default:queued" json:"status"`
	Error     string         `gorm:"type:text" json:"error"`
	StartedAt *time.Time     `json:"startedAt"`
	EndedAt   *time.Time     `json:"endedAt"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type TaskLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    uint      `gorm:"index;not null" json:"taskId"`
	Level     string    `gorm:"size:16;not null;default:info" json:"level"`
	Message   string    `gorm:"type:text;not null" json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type OperationLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    *uint     `gorm:"index" json:"userId"`
	Action    string    `gorm:"size:120;not null" json:"action"`
	Resource  string    `gorm:"size:120" json:"resource"`
	Metadata  string    `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt time.Time `json:"createdAt"`
}
