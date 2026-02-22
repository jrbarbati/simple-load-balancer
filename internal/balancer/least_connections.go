package balancer

import (
	"load-balancer/internal/backend"
	"math"
)

type LeastConnections struct {
}

func NewLeastConnections() *LeastConnections {
	return &LeastConnections{}
}

func (lc *LeastConnections) NextBackend(backends []*backend.Backend) (*backend.Backend, error) {
	if len(backends) <= 0 {
		return nil, NoRegisteredBackends
	}

	backendWithMinConnections := minActiveConnections(backends)

	if backendWithMinConnections == nil {
		return nil, NoHealthyBackends
	}

	return backendWithMinConnections, nil
}

func minActiveConnections(backends []*backend.Backend) *backend.Backend {
	minSoFar := int32(math.MaxInt32)
	var selectedBackend *backend.Backend

	for _, be := range backends {
		if !be.IsHealthy() {
			continue
		}

		if be.ActiveConnections() >= minSoFar {
			continue
		}

		minSoFar = be.ActiveConnections()
		selectedBackend = be
	}

	return selectedBackend
}
