package converter

import "testing"

func TestParseKnownShareLinkValuesVLESSReality(t *testing.T) {
	values := ParseKnownShareLinkValues("vless://8660626b-d84f-4649-b206-7c065b61cf08@wj.tikle.vip:13348?type=tcp&encryption=none&security=reality&pbk=A5pF3kaVzVH2PR0Dhq2M1G28HyeHsQqus3gfHeavFyw&fp=chrome&sni=tesla.com&sid=ce4f61&spx=%2F&flow=xtls-rprx-vision#Reality")
	if values["uuid"] != "8660626b-d84f-4649-b206-7c065b61cf08" ||
		values["network"] != "tcp" ||
		values["security"] != "reality" ||
		values["tls"] != "true" ||
		values["servername"] != "tesla.com" ||
		values["pbk"] != "A5pF3kaVzVH2PR0Dhq2M1G28HyeHsQqus3gfHeavFyw" ||
		values["sid"] != "ce4f61" {
		t.Fatalf("unexpected vless reality values: %#v", values)
	}
}

func TestParseKnownShareLinkValuesVMess(t *testing.T) {
	values := ParseKnownShareLinkValues("vmess://eyJ2IjoiMiIsInBzIjoiVk1lc3MiLCJhZGQiOiJleGFtcGxlLmNvbSIsInBvcnQiOiI0NDMiLCJpZCI6InV1aWQtdmFsdWUiLCJhaWQiOiIwIiwic2N5IjoiYXV0byIsIm5ldCI6IndzIiwidHlwZSI6Im5vbmUiLCJob3N0Ijoid3MuZXhhbXBsZS5jb20iLCJwYXRoIjoiL3dzIiwidGxzIjoidGxzIiwic25pIjoidGxzLmV4YW1wbGUuY29tIn0=")
	if values["uuid"] != "uuid-value" ||
		values["alterId"] != "0" ||
		values["cipher"] != "auto" ||
		values["network"] != "ws" ||
		values["servername"] != "tls.example.com" ||
		values["host"] != "ws.example.com" ||
		values["path"] != "/ws" {
		t.Fatalf("unexpected vmess values: %#v", values)
	}
}

func TestParseKnownShareLinkValuesLeavesUnsupportedToFallback(t *testing.T) {
	for _, rawLink := range []string{
		"anytls://password@example.com:8443?peer=addons.mozilla.org&hpkp=pin#AnyTLS",
		"socks5://user:pass@example.com:1080#Socks",
	} {
		if values := ParseKnownShareLinkValues(rawLink); values != nil {
			t.Fatalf("expected unsupported link to use fallback, got %#v for %s", values, rawLink)
		}
	}
}
