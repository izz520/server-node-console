package argosbx

import (
	"strings"
	"testing"
)

func TestBuildInstallCommandWrapsScriptWithRemoteBootstrap(t *testing.T) {
	command, varName, err := BuildInstallCommand(InstallParams{
		Protocol:   "AnyTLS",
		Port:       28443,
		UUID:       "uuid-value",
		NamePrefix: "AnyTLS O'Neil Node",
	})
	if err != nil {
		t.Fatalf("BuildInstallCommand returned error: %v", err)
	}
	if varName != "anpt" {
		t.Fatalf("expected var name anpt, got %q", varName)
	}
	if !strings.HasPrefix(command, "sh -lc ") {
		t.Fatalf("expected command to run through sh -lc bootstrap, got %q", command)
	}
	for _, want := range []string{
		"command -v bash",
		"command -v curl",
		"install_pkg",
		"exec bash -lc",
		"anpt=",
		"name=",
		"uuid=",
		"curl -Ls https://raw.githubusercontent.com/yonggekkk/argosbx/main/argosbx.sh",
	} {
		if !strings.Contains(command, want) {
			t.Fatalf("expected command to contain %q, got %q", want, command)
		}
	}
	if !strings.Contains(command, "'\\''") {
		t.Fatalf("expected nested single quotes to be escaped, got %q", command)
	}
	if strings.Contains(command, "{;") || strings.Contains(command, ";;") {
		t.Fatalf("expected POSIX sh-compatible function syntax, got %q", command)
	}
}

func TestBuildUninstallCommandWrapsScriptWithRemoteBootstrap(t *testing.T) {
	command := BuildUninstallCommand()
	if !strings.HasPrefix(command, "sh -lc ") {
		t.Fatalf("expected uninstall command to run through sh -lc bootstrap, got %q", command)
	}
	if !strings.Contains(command, "exec bash -lc") || !strings.Contains(command, "argosbx.sh") || !strings.Contains(command, " del") {
		t.Fatalf("expected uninstall command to call argosbx delete, got %q", command)
	}
}

func TestShellSingleQuoteEscapesPOSIXSingleQuotes(t *testing.T) {
	got := shellSingleQuote("AnyTLS O'Neil Node")
	want := "'AnyTLS O'\\''Neil Node'"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
