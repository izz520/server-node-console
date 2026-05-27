package argosbx

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const MainScript = "bash <(curl -Ls https://raw.githubusercontent.com/yonggekkk/argosbx/main/argosbx.sh)"

var protocolVariables = map[string]string{
	"AnyTLS":                         "anpt",
	"Any-reality":                    "arpt",
	"Vless-xhttp-reality-vision-enc": "xhpt",
	"Vless-tcp-reality-vision":       "vlpt",
	"Vless-xhttp-vision-enc":         "vxpt",
	"Vless-ws-vision-enc":            "vwpt",
	"Shadowsocks-2022":               "sspt",
	"Hysteria2":                      "hypt",
	"Tuic":                           "tupt",
	"Socks5":                         "sopt",
	"Vmess-ws":                       "vmpt",
	"Argo临时隧道":                       "vmpt",
	"Argo固定隧道":                       "vmpt",
	"Argo 临时隧道":                      "vmpt",
	"Argo 固定隧道":                      "vmpt",
}

type InstallParams struct {
	Protocol      string
	Port          int
	UUID          string
	RealityDomain string
	CDNDomain     string
	ArgoMode      string
	ArgoDomain    string
	ArgoToken     string
	NamePrefix    string
}

func BuildInstallCommand(params InstallParams) (string, string, error) {
	command, err := BuildInstallCommandSet([]InstallParams{params})
	if err != nil {
		return "", "", err
	}
	varName, ok := protocolVariables[params.Protocol]
	if !ok {
		return "", "", fmt.Errorf("unsupported protocol: %s", params.Protocol)
	}
	return command, varName, nil
}

func BuildInstallCommandSet(params []InstallParams) (string, error) {
	if len(params) == 0 {
		return "", errors.New("install params are required")
	}
	values := map[string]string{}
	usedVars := map[string]string{}
	for _, param := range params {
		varName, ok := protocolVariables[param.Protocol]
		if !ok {
			return "", fmt.Errorf("unsupported protocol: %s", param.Protocol)
		}
		if usedProtocol, ok := usedVars[varName]; ok {
			return "", fmt.Errorf("argosbx cannot install multiple %s nodes in one server run; %s and %s both use %s", param.Protocol, usedProtocol, param.Protocol, varName)
		}
		usedVars[varName] = param.Protocol
		values[varName] = ""
		if param.Port > 0 {
			values[varName] = fmt.Sprintf("%d", param.Port)
		}
		setFirst(values, "uuid", param.UUID)
		setFirst(values, "reym", param.RealityDomain)
		setFirst(values, "cdnym", param.CDNDomain)
		if len(params) == 1 {
			setFirst(values, "name", param.NamePrefix)
		}
		if strings.Contains(param.Protocol, "Argo") || param.ArgoMode != "" {
			setFirst(values, "argo", varName)
		}
		setFirst(values, "agn", param.ArgoDomain)
		setFirst(values, "agk", param.ArgoToken)
	}

	prefix := renderVars(values)
	return remoteShellCommand(strings.TrimSpace(prefix + " " + MainScript + " rep")), nil
}

func setFirst(values map[string]string, key string, value string) {
	if value == "" {
		return
	}
	if _, exists := values[key]; exists {
		return
	}
	values[key] = value
}

func BuildUninstallCommand() string {
	return remoteShellCommand(MainScript + " del")
}

func VarNameForProtocol(protocol string) (string, error) {
	value, ok := protocolVariables[protocol]
	if !ok {
		return "", errors.New("unsupported protocol")
	}
	return value, nil
}

func renderVars(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, shellSingleQuote(values[key])))
	}
	return strings.Join(parts, " ")
}

func remoteShellCommand(script string) string {
	bootstrap := strings.Join([]string{
		"set -e",
		"install_pkg() { if command -v apt-get >/dev/null 2>&1; then export DEBIAN_FRONTEND=noninteractive; apt-get update; apt-get install -y \"$@\"; elif command -v apk >/dev/null 2>&1; then apk add --no-cache \"$@\"; elif command -v dnf >/dev/null 2>&1; then dnf install -y \"$@\"; elif command -v yum >/dev/null 2>&1; then yum install -y \"$@\"; elif command -v pacman >/dev/null 2>&1; then pacman -Sy --noconfirm \"$@\"; else echo \"missing package manager for required command(s): $*\" >&2; exit 127; fi; }",
		"missing=\"\"",
		"command -v bash >/dev/null 2>&1 || missing=\"$missing bash\"",
		"command -v curl >/dev/null 2>&1 || missing=\"$missing curl\"",
		"if [ -n \"$missing\" ]; then install_pkg $missing; fi",
		"exec bash -lc " + shellSingleQuote(script),
	}, "; ")
	return "sh -lc " + shellSingleQuote(bootstrap)
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
