package argosbx

import (
	"fmt"
	"strings"
)

var installFailureMarkers = []string{
	"安装失败",
	"install failed",
}

var installSuccessMarkers = []string{
	"Argosbx脚本输出节点配置如下",
	"聚合节点信息，请进入",
}

func DetectInstallFailure(output string) error {
	normalized := strings.ToLower(output)
	for _, marker := range installFailureMarkers {
		if strings.Contains(normalized, strings.ToLower(marker)) {
			return fmt.Errorf("argosbx install reported failure: %s", marker)
		}
	}
	for _, marker := range installSuccessMarkers {
		if strings.Contains(output, marker) {
			return nil
		}
	}
	if strings.Contains(output, "Argosbx脚本已安装") {
		return fmt.Errorf("argosbx install did not run protocol update; script only reported existing installation")
	}
	return fmt.Errorf("argosbx install did not output node configuration")
}

func ExtractShareLink(output string, protocol string) string {
	schemes := shareLinkSchemes(protocol)
	if len(schemes) == 0 {
		return ""
	}
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		for _, scheme := range schemes {
			prefix := scheme + "://"
			index := strings.Index(strings.ToLower(line), prefix)
			if index >= 0 {
				return strings.TrimSpace(strings.Trim(line[index:], `"'<>，。；;、`))
			}
		}
	}
	return ""
}

func shareLinkSchemes(protocol string) []string {
	value := strings.ToLower(strings.TrimSpace(protocol))
	switch {
	case strings.HasPrefix(value, "vless") || strings.Contains(value, "reality"):
		return []string{"vless"}
	case strings.HasPrefix(value, "vmess") || strings.Contains(value, "argo"):
		return []string{"vmess"}
	case strings.HasPrefix(value, "shadowsocks"):
		return []string{"ss"}
	case strings.HasPrefix(value, "hysteria2"):
		return []string{"hysteria2", "hy2"}
	case strings.HasPrefix(value, "tuic"):
		return []string{"tuic"}
	case strings.HasPrefix(value, "socks"):
		return []string{"socks5", "socks"}
	case strings.HasPrefix(value, "anytls"):
		return []string{"anytls"}
	default:
		return nil
	}
}
