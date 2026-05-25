package argosbx

import "testing"

func TestDetectInstallFailureWithChineseFailureMarker(t *testing.T) {
	output := "Argosbx脚本进程未启动，安装失败"

	err := DetectInstallFailure(output)
	if err == nil {
		t.Fatal("expected install failure to be detected")
	}
}

func TestDetectInstallFailureWithSuccessfulOutput(t *testing.T) {
	output := "Argosbx脚本进程启动成功\ninstall task completed"

	err := DetectInstallFailure(output)
	if err != nil {
		t.Fatalf("expected successful output to pass, got %v", err)
	}
}
