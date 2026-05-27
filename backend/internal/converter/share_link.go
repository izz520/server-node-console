package converter

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

// ParseKnownShareLinkValues normalizes share links using the protocol model
// from sublink-worker. Protocols not covered by sublink-worker's direct URL
// parsers, such as AnyTLS and Socks5, intentionally return nil so callers can
// keep their existing richer fallback logic.
func ParseKnownShareLinkValues(rawLink string) map[string]string {
	rawLink = strings.TrimSpace(rawLink)
	if rawLink == "" {
		return nil
	}

	if strings.HasPrefix(strings.ToLower(rawLink), "vmess://") {
		return parseVMessValues(rawLink)
	}

	parsed, err := url.Parse(rawLink)
	if err != nil || parsed.Scheme == "" {
		return nil
	}

	switch strings.ToLower(parsed.Scheme) {
	case "ss":
		return parseShadowsocksValues(rawLink, parsed)
	case "vless":
		return parseVLESSValues(parsed)
	case "trojan":
		return parseTrojanValues(parsed)
	case "hysteria", "hysteria2", "hy2":
		return parseHysteriaValues(parsed)
	case "tuic":
		return parseTUICValues(parsed)
	default:
		return nil
	}
}

func parseVMessValues(rawLink string) map[string]string {
	encoded := strings.TrimPrefix(rawLink, "vmess://")
	decoded := decodeBase64String(encoded)
	if decoded == "" {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(decoded), &payload); err != nil {
		return nil
	}

	values := map[string]string{}
	copyAnyString(values, payload, "name", "ps")
	copyAnyString(values, payload, "address", "add")
	copyAnyString(values, payload, "port", "port")
	copyAnyString(values, payload, "uuid", "id")
	copyAnyString(values, payload, "alterId", "aid")
	copyAnyString(values, payload, "cipher", "scy")
	copyAnyString(values, payload, "network", "net")
	copyAnyString(values, payload, "type", "net")
	copyAnyString(values, payload, "headerType", "type")
	copyAnyString(values, payload, "tls", "tls")
	copyAnyString(values, payload, "security", "tls")
	copyAnyString(values, payload, "servername", "sni")
	copyAnyString(values, payload, "sni", "sni")
	copyAnyString(values, payload, "host", "host")
	copyAnyString(values, payload, "path", "path")
	copyAnyString(values, payload, "alpn", "alpn")
	copyAnyString(values, payload, "fp", "fp")
	return emptyToNil(values)
}

func parseShadowsocksValues(rawLink string, parsed *url.URL) map[string]string {
	values := queryValues(parsed)
	mainPart := strings.TrimPrefix(rawLink, "ss://")
	if index := strings.IndexAny(mainPart, "?#"); index >= 0 {
		mainPart = mainPart[:index]
	}

	if parsed.User != nil {
		username := strings.TrimSpace(parsed.User.Username())
		password, hasPassword := parsed.User.Password()
		if hasPassword {
			values["cipher"] = username
			values["password"] = strings.TrimSpace(password)
			return emptyToNil(values)
		}
		if decoded := decodeBase64String(username); decoded != "" {
			setCipherPassword(values, decoded)
		}
		return emptyToNil(values)
	}

	if decoded := decodeBase64String(mainPart); decoded != "" {
		if beforeHost, _, ok := strings.Cut(decoded, "@"); ok {
			setCipherPassword(values, beforeHost)
		} else {
			setCipherPassword(values, decoded)
		}
	}
	return emptyToNil(values)
}

func parseVLESSValues(parsed *url.URL) map[string]string {
	values := queryValues(parsed)
	if parsed.User != nil {
		values["uuid"] = strings.TrimSpace(parsed.User.Username())
	}
	if network := firstNonEmpty(values["type"], values["network"]); network != "" {
		values["network"] = network
	}
	if sni := firstNonEmpty(values["sni"], values["servername"], values["server_name"]); sni != "" {
		values["servername"] = sni
	}
	if strings.EqualFold(values["security"], "reality") {
		values["tls"] = "true"
		if firstNonEmpty(values["fp"], values["client-fingerprint"], values["fingerprint"]) == "" {
			values["fp"] = "chrome"
		}
	}
	return emptyToNil(values)
}

func parseTrojanValues(parsed *url.URL) map[string]string {
	values := queryValues(parsed)
	if parsed.User != nil {
		values["password"] = strings.TrimSpace(parsed.User.Username())
	}
	if network := firstNonEmpty(values["type"], values["network"]); network != "" {
		values["network"] = network
	}
	if sni := firstNonEmpty(values["sni"], values["servername"], values["server_name"]); sni != "" {
		values["servername"] = sni
	}
	if firstNonEmpty(values["security"], values["tls"]) == "" {
		values["tls"] = "true"
	}
	return emptyToNil(values)
}

func parseHysteriaValues(parsed *url.URL) map[string]string {
	values := queryValues(parsed)
	if parsed.User != nil {
		values["password"] = strings.TrimSpace(parsed.User.Username())
	}
	if auth := firstNonEmpty(values["auth"], values["auth-str"], values["auth_str"]); auth != "" && values["password"] == "" {
		values["password"] = auth
	}
	if sni := firstNonEmpty(values["sni"], values["peer"], values["servername"], values["server_name"]); sni != "" {
		values["servername"] = sni
	}
	if insecure := firstNonEmpty(values["insecure"], values["allowInsecure"], values["allow_insecure"], values["skip-cert-verify"], values["skip_cert_verify"]); insecure != "" {
		values["skip-cert-verify"] = insecure
	}
	return emptyToNil(values)
}

func parseTUICValues(parsed *url.URL) map[string]string {
	values := queryValues(parsed)
	if parsed.User != nil {
		username := strings.TrimSpace(parsed.User.Username())
		password, hasPassword := parsed.User.Password()
		if hasPassword {
			values["uuid"] = username
			values["password"] = strings.TrimSpace(password)
		} else if uuid, password, ok := strings.Cut(username, ":"); ok {
			values["uuid"] = strings.TrimSpace(uuid)
			values["password"] = strings.TrimSpace(password)
		} else {
			values["token"] = username
		}
	}
	if sni := firstNonEmpty(values["sni"], values["servername"], values["server_name"]); sni != "" {
		values["servername"] = sni
	}
	if congestion := firstNonEmpty(values["congestion_control"], values["congestion-controller"], values["congestion"]); congestion != "" {
		values["congestion-controller"] = congestion
	}
	if insecure := firstNonEmpty(values["insecure"], values["allowInsecure"], values["allow_insecure"], values["skip-cert-verify"], values["skip_cert_verify"]); insecure != "" {
		values["skip-cert-verify"] = insecure
	}
	return emptyToNil(values)
}

func queryValues(parsed *url.URL) map[string]string {
	values := map[string]string{}
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

func setCipherPassword(values map[string]string, value string) {
	cipher, password, ok := strings.Cut(value, ":")
	if !ok {
		return
	}
	if cipher = strings.TrimSpace(cipher); cipher != "" {
		values["cipher"] = cipher
	}
	if password = strings.TrimSpace(password); password != "" {
		values["password"] = password
	}
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func emptyToNil(values map[string]string) map[string]string {
	for key, value := range values {
		if strings.TrimSpace(value) == "" {
			delete(values, key)
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
