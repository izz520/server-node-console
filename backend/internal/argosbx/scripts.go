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
	varName, ok := protocolVariables[params.Protocol]
	if !ok {
		return "", "", fmt.Errorf("unsupported protocol: %s", params.Protocol)
	}

	values := map[string]string{
		varName: "",
	}
	if params.Port > 0 {
		values[varName] = fmt.Sprintf("%d", params.Port)
	}
	if params.UUID != "" {
		values["uuid"] = params.UUID
	}
	if params.RealityDomain != "" {
		values["reym"] = params.RealityDomain
	}
	if params.CDNDomain != "" {
		values["cdnym"] = params.CDNDomain
	}
	if params.NamePrefix != "" {
		values["name"] = params.NamePrefix
	}
	if strings.Contains(params.Protocol, "Argo") || params.ArgoMode != "" {
		values["argo"] = varName
	}
	if params.ArgoDomain != "" {
		values["agn"] = params.ArgoDomain
	}
	if params.ArgoToken != "" {
		values["agk"] = params.ArgoToken
	}

	prefix := renderVars(values)
	return strings.TrimSpace(prefix + " " + MainScript), varName, nil
}

func BuildUninstallCommand() string {
	return MainScript + " del"
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
		parts = append(parts, fmt.Sprintf(`%s="%s"`, key, shellEscape(values[key])))
	}
	return strings.Join(parts, " ")
}

func shellEscape(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}
