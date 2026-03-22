//go:build linux

package agentplatform

import (
	"github.com/labtether/labtether-linux/internal/agentcore"
	"github.com/labtether/labtether-linux/internal/agentplatform/linux"
	"github.com/labtether/labtether-linux/pkg/platforms"
)

func init() {
	providerFactories[platforms.Linux] = func(assetID, source string) agentcore.TelemetryProvider {
		return linux.New(assetID, source)
	}
}
