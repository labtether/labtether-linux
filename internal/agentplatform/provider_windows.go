//go:build windows

package agentplatform

import (
	"github.com/labtether/labtether-linux/internal/agentcore"
	"github.com/labtether/labtether-linux/internal/agentplatform/windows"
	"github.com/labtether/labtether-linux/pkg/platforms"
)

func init() {
	providerFactories[platforms.Windows] = func(assetID, source string) agentcore.TelemetryProvider {
		return windows.New(assetID, source)
	}
}
