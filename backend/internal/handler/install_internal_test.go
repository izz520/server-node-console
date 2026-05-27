package handler

import (
	"encoding/json"
	"testing"

	"server-sing-box-2/backend/internal/domain"
	"server-sing-box-2/backend/internal/security"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUpdateInstalledTargetsFromOutputPersistsOutputUUID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.ProtocolNode{}); err != nil {
		t.Fatalf("migrate protocol nodes: %v", err)
	}
	node := domain.ProtocolNode{
		UserID:        1,
		Name:          "Reality",
		Protocol:      "Vless-tcp-reality-vision",
		ListenPort:    10324,
		InstallMethod: domain.NodeInstallMethodSystem,
		Status:        domain.NodeStatusInstalling,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	h := &Handler{db: db, encryptionKey: "test-encryption-key"}
	outputUUID := "b450d832-1681-466c-94d2-ea33b96fb6da"
	h.updateInstalledTargetsFromOutput(0, []installTarget{{
		NodeID: node.ID,
		Req: installNodeRequest{
			Name:     "Reality",
			Protocol: "Vless-tcp-reality-vision",
			Port:     10324,
			UUID:     "6ea1234e-9961-4bcf-8618-45f71b23adce",
		},
	}}, "vless://"+outputUUID+"@38.55.108.55:10324?security=reality&sni=apple.com#Reality")

	var stored domain.ProtocolNode
	if err := db.First(&stored, node.ID).Error; err != nil {
		t.Fatalf("load node: %v", err)
	}
	encryptor, err := security.NewEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	plain, err := encryptor.Decrypt(stored.EncryptedProtocolJSON)
	if err != nil {
		t.Fatalf("decrypt node config: %v", err)
	}
	config := encryptedNodeConfig{}
	if err := json.Unmarshal([]byte(plain), &config); err != nil {
		t.Fatalf("decode node config: %v", err)
	}
	values := map[string]string{}
	if err := json.Unmarshal([]byte(config.Sensitive), &values); err != nil {
		t.Fatalf("decode sensitive values: %v", err)
	}
	if values["uuid"] != outputUUID {
		t.Fatalf("expected output UUID %q to be persisted, got %q", outputUUID, values["uuid"])
	}
}
