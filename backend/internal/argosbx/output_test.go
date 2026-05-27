package argosbx

import (
	"strings"
	"testing"
)

func TestDetectInstallFailureWithChineseFailureMarker(t *testing.T) {
	output := "Argosbx脚本进程未启动，安装失败"

	err := DetectInstallFailure(output)
	if err == nil {
		t.Fatal("expected install failure to be detected")
	}
}

func TestDetectInstallFailureWithSuccessfulOutput(t *testing.T) {
	output := "Argosbx脚本输出节点配置如下\n聚合节点信息，请进入 /root/agsbx/jhsub.txt 文件目录查看"

	err := DetectInstallFailure(output)
	if err != nil {
		t.Fatalf("expected successful output to pass, got %v", err)
	}
}

func TestDetectInstallFailureWhenScriptOnlyReportsExistingInstall(t *testing.T) {
	output := "Argosbx脚本已安装\n=========当前三大内核运行状态=========\nSing-box：运行中"

	err := DetectInstallFailure(output)
	if err == nil {
		t.Fatal("expected status-only output to be detected as failed install")
	}
}

func TestDetectInstallFailureWithoutNodeConfiguration(t *testing.T) {
	output := "Sing-box：运行中\ninstall task completed"

	err := DetectInstallFailure(output)
	if err == nil {
		t.Fatal("expected missing node configuration to be detected")
	}
}

func TestExtractShareLinkForVLESSReality(t *testing.T) {
	link := "vless://c0464a28-9013-4e71-b21c-feb8db08dd8e@38.55.108.55:48607?encryption=none&flow=xtls-rprx-vision&security=reality&sni=apple.com&fp=chrome&pbk=Sjwj_5APjh2rKP0HC1anVN2-Ey1LtjLNq16VPn_r4Bg&sid=55cde0a4&type=tcp&headerType=none#🇺🇸 LazyCat-VMISS-Reality"
	output := "Argosbx脚本输出节点配置如下：\n" + link + "\n聚合节点信息，请进入 /root/agsbx/jhsub.txt"

	got := ExtractShareLink(output, "Vless-tcp-reality-vision")
	if got != link {
		t.Fatalf("expected extracted link %q, got %q", link, got)
	}
}

func TestExtractShareLinkChoosesProtocolScheme(t *testing.T) {
	output := strings.Join([]string{
		"vmess://ignored",
		"ss://YWVzLTEyOC1nY206cGFzcw@example.com:8388#SS",
	}, "\n")

	got := ExtractShareLink(output, "Shadowsocks-2022")
	if !strings.HasPrefix(got, "ss://") {
		t.Fatalf("expected shadowsocks link, got %q", got)
	}
}

func TestExtractShareLinksReturnsAllProtocolLinks(t *testing.T) {
	output := strings.Join([]string{
		"vless://uuid@example.com:443?security=reality#Reality",
		"anytls://password@example.com:8443?insecure=0#AnyTLS",
		"anytls://password2@example.com:9443?insecure=0#AnyTLS2",
	}, "\n")

	got := ExtractShareLinks(output, "AnyTLS")
	if len(got) != 2 || !strings.Contains(got[0], ":8443") || !strings.Contains(got[1], ":9443") {
		t.Fatalf("expected two anytls links, got %#v", got)
	}
}
