package docker

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labtether/labtether-linux/pkg/securityruntime"
)

// The session context owns follows and duplex upgrades. Keep the shared
// client's finite timeout for ordinary Docker API requests.
func (c *dockerClient) doStreamRequest(req *http.Request) (*http.Response, error) {
	if req == nil || req.Context().Done() == nil {
		return nil, fmt.Errorf("Docker stream requires a cancellable context")
	}
	client := *c.httpClient
	client.Timeout = 0
	if strings.TrimSpace(c.unixPath) != "" {
		return c.doRequest(&client, req)
	}
	return securityruntime.DoOutboundStreamRequest(&client, req)
}
