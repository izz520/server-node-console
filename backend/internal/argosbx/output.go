package argosbx

import (
	"fmt"
	"strings"
)

var installFailureMarkers = []string{
	"安装失败",
	"install failed",
}

func DetectInstallFailure(output string) error {
	normalized := strings.ToLower(output)
	for _, marker := range installFailureMarkers {
		if strings.Contains(normalized, strings.ToLower(marker)) {
			return fmt.Errorf("argosbx install reported failure: %s", marker)
		}
	}
	return nil
}
