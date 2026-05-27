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
		UserID:                 1,
		Name:                   "Reality",
		Protocol:               "Vless-tcp-reality-vision",
		ListenPort:             10324,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstalling,
		SubscriptionConfigJSON: `{"address":"old.example.com","port":443,"generatedFrom":"argosbx"}`,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	h := &Handler{db: db, encryptionKey: "test-encryption-key"}
	outputUUID := "b450d832-1681-466c-94d2-ea33b96fb6da"
	rawLink := "vless://" + outputUUID + "@38.55.108.55:10324?security=reality&sni=apple.com&pbk=public-key&sid=short-id#Reality"
	h.updateInstalledTargetsFromOutput(0, []installTarget{{
		NodeID: node.ID,
		Req: installNodeRequest{
			Name:     "Reality",
			Protocol: "Vless-tcp-reality-vision",
			Port:     10324,
			UUID:     "6ea1234e-9961-4bcf-8618-45f71b23adce",
		},
	}}, rawLink)

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
	if values["security"] != "reality" || values["sni"] != "apple.com" || values["pbk"] != "public-key" || values["sid"] != "short-id" {
		t.Fatalf("expected output reality params to be persisted, got %#v", values)
	}
	subscriptionConfig := nodeConfig{}
	if err := json.Unmarshal([]byte(stored.SubscriptionConfigJSON), &subscriptionConfig); err != nil {
		t.Fatalf("decode subscription config: %v", err)
	}
	if subscriptionConfig.Address != "old.example.com" || subscriptionConfig.Port != 10324 || subscriptionConfig.RawLink != rawLink {
		t.Fatalf("expected system address with port and raw link from output, got %+v", subscriptionConfig)
	}
}

func TestUpdateInstalledTargetsFromOutputPersistsShadowsocksOutput(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.ProtocolNode{}); err != nil {
		t.Fatalf("migrate protocol nodes: %v", err)
	}
	encryptor, err := security.NewEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	oldEncrypted, err := encryptor.Encrypt(`{"sensitive":"{\"password\":\"old-password\",\"cipher\":\"2022-blake3-aes-128-gcm\"}"}`)
	if err != nil {
		t.Fatalf("encrypt old config: %v", err)
	}
	node := domain.ProtocolNode{
		UserID:                 1,
		Name:                   "SS",
		Protocol:               "Shadowsocks-2022",
		ListenPort:             56603,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstalling,
		EncryptedProtocolJSON:  oldEncrypted,
		SubscriptionConfigJSON: `{"address":"ft-us.lazycat.cv","port":56603,"generatedFrom":"argosbx"}`,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	h := &Handler{db: db, encryptionKey: "test-encryption-key"}
	rawLink := "ss://MjAyMi1ibGFrZTMtYWVzLTEyOC1nY206TDZ5aDVQVVdkYis1MnJ0MFhBb0t6dz09QDQ3LjE0Ny4xOC4yOjI1MTIz#SS"
	h.updateInstalledTargetsFromOutput(0, []installTarget{{
		NodeID: node.ID,
		Req: installNodeRequest{
			Name:     "SS",
			Protocol: "Shadowsocks-2022",
			Port:     56603,
			UUID:     "old-password",
		},
	}}, rawLink)

	var stored domain.ProtocolNode
	if err := db.First(&stored, node.ID).Error; err != nil {
		t.Fatalf("load node: %v", err)
	}
	if stored.ListenPort != 25123 {
		t.Fatalf("expected listen port from output, got %d", stored.ListenPort)
	}
	config := nodeConfig{}
	if err := json.Unmarshal([]byte(stored.SubscriptionConfigJSON), &config); err != nil {
		t.Fatalf("decode subscription config: %v", err)
	}
	if config.Address != "ft-us.lazycat.cv" || config.Port != 25123 || config.RawLink != rawLink {
		t.Fatalf("expected system address with subscription port from output, got %+v", config)
	}
	plain, err := encryptor.Decrypt(stored.EncryptedProtocolJSON)
	if err != nil {
		t.Fatalf("decrypt node config: %v", err)
	}
	encryptedConfig := encryptedNodeConfig{}
	if err := json.Unmarshal([]byte(plain), &encryptedConfig); err != nil {
		t.Fatalf("decode encrypted config: %v", err)
	}
	values := map[string]string{}
	if err := json.Unmarshal([]byte(encryptedConfig.Sensitive), &values); err != nil {
		t.Fatalf("decode sensitive values: %v", err)
	}
	if values["password"] != "L6yh5PUWdb+52rt0XAoKzw==" || values["cipher"] != "2022-blake3-aes-128-gcm" {
		t.Fatalf("expected shadowsocks sensitive values from output, got %#v", values)
	}
}

func TestUpdateInstalledTargetsFromOutputPersistsGenericRawLinkValues(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.ProtocolNode{}); err != nil {
		t.Fatalf("migrate protocol nodes: %v", err)
	}
	node := domain.ProtocolNode{
		UserID:                 1,
		Name:                   "AnyTLS",
		Protocol:               "AnyTLS",
		ListenPort:             8443,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstalling,
		SubscriptionConfigJSON: `{"address":"old.example.com","port":8443,"generatedFrom":"argosbx"}`,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	h := &Handler{db: db, encryptionKey: "test-encryption-key"}
	rawLink := "anytls://new-password@203.0.113.10:24443?peer=addons.mozilla.org&udp=1&hpkp=pin-value#AnyTLS"
	h.updateInstalledTargetsFromOutput(0, []installTarget{{
		NodeID: node.ID,
		Req: installNodeRequest{
			Name:     "AnyTLS",
			Protocol: "AnyTLS",
			Port:     8443,
			UUID:     "old-password",
		},
	}}, rawLink)

	var stored domain.ProtocolNode
	if err := db.First(&stored, node.ID).Error; err != nil {
		t.Fatalf("load node: %v", err)
	}
	config := nodeConfig{}
	if err := json.Unmarshal([]byte(stored.SubscriptionConfigJSON), &config); err != nil {
		t.Fatalf("decode subscription config: %v", err)
	}
	if config.Address != "old.example.com" || config.Port != 24443 || config.RawLink != rawLink {
		t.Fatalf("expected system address with endpoint port from output, got %+v", config)
	}
	encryptor, err := security.NewEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	plain, err := encryptor.Decrypt(stored.EncryptedProtocolJSON)
	if err != nil {
		t.Fatalf("decrypt node config: %v", err)
	}
	encryptedConfig := encryptedNodeConfig{}
	if err := json.Unmarshal([]byte(plain), &encryptedConfig); err != nil {
		t.Fatalf("decode encrypted config: %v", err)
	}
	values := map[string]string{}
	if err := json.Unmarshal([]byte(encryptedConfig.Sensitive), &values); err != nil {
		t.Fatalf("decode sensitive values: %v", err)
	}
	if values["password"] != "new-password" || values["uuid"] != "new-password" || values["peer"] != "addons.mozilla.org" || values["hpkp"] != "pin-value" || values["udp"] != "1" {
		t.Fatalf("expected all anytls values from output, got %#v", values)
	}
}

func TestSubscriptionNodeViewPrefersSystemRawLinkValues(t *testing.T) {
	encryptor, err := security.NewEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	rawLink := "ss://MjAyMi1ibGFrZTMtYWVzLTEyOC1nY206TDZ5aDVQVVdkYis1MnJ0MFhBb0t6dz09QDQ3LjE0Ny4xOC4yOjI1MTIz#SS"
	encrypted, err := encryptor.Encrypt(`{"rawLink":"` + rawLink + `","sensitive":"{\"password\":\"old-password\",\"cipher\":\"2022-blake3-aes-128-gcm\"}"}`)
	if err != nil {
		t.Fatalf("encrypt config: %v", err)
	}
	node := domain.ProtocolNode{
		Name:                   "SS",
		Protocol:               "Shadowsocks-2022",
		ListenPort:             56603,
		InstallMethod:          domain.NodeInstallMethodSystem,
		Status:                 domain.NodeStatusInstallOK,
		EncryptedProtocolJSON:  encrypted,
		SubscriptionConfigJSON: `{"address":"ft-us.lazycat.cv","port":56603,"generatedFrom":"argosbx"}`,
	}

	view := subscriptionNodeViewFromNode(node, nil, map[string]string{
		encryptedRawLinkKey: rawLink,
		"password":          "old-password",
		"cipher":            "2022-blake3-aes-128-gcm",
	})

	if view.Address != "ft-us.lazycat.cv" || view.Port != 25123 || view.Password != "L6yh5PUWdb+52rt0XAoKzw==" {
		t.Fatalf("expected view to prefer raw link values, got %+v", view)
	}
}

func TestShareLineUsesSystemAddressWithRawLinkParams(t *testing.T) {
	view := subscriptionNodeView{
		Name:     "LazyCat-TW-ISP-Anytls",
		Protocol: "AnyTLS",
		Address:  "hinet-1.lazycat.cv",
		Port:     30037,
		RawLink:  "anytls://password@111.253.47.214:30037?peer=addons.mozilla.org&udp=1&hpkp=pin#LazyCat-TW-ISP-Anytls",
	}

	got := view.shareLine()
	want := "anytls://password@hinet-1.lazycat.cv:30037?peer=addons.mozilla.org&udp=1&hpkp=pin#LazyCat-TW-ISP-Anytls"
	if got != want {
		t.Fatalf("expected share line %q, got %q", want, got)
	}
}
