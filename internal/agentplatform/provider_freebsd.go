//go:build freebsd

package agentplatform

import (
	"github.com/labtether/labtether-linux/internal/agentcore"
	"github.com/labtether/labtether-linux/internal/agentplatform/freebsd"
	"github.com/labtether/labtether-linux/pkg/platforms"
)

func init() {
	providerFactories[platforms.FreeBSD] = func(assetID, source string) agentcore.TelemetryProvider {
		return freebsd.New(assetID, source)
	}
}
