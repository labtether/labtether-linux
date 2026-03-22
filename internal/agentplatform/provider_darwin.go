//go:build darwin

package agentplatform

import (
	"github.com/labtether/labtether-linux/internal/agentcore"
	"github.com/labtether/labtether-linux/internal/agentplatform/darwin"
	"github.com/labtether/labtether-linux/pkg/platforms"
)

func init() {
	providerFactories[platforms.Darwin] = func(assetID, source string) agentcore.TelemetryProvider {
		return darwin.New(assetID, source)
	}
}
