package balancer

import (
	"load-balancer/internal/backend"
	"sync/atomic"
)

type RoundRobin struct {
	routeToIndex *atomic.Uint32
}

func NewRoundRobin() *RoundRobin {
	routeToIndex := new(atomic.Uint32)
	routeToIndex.Store(0)

	return &RoundRobin{routeToIndex}
}

func (rr *RoundRobin) NextBackend(backends []*backend.Backend) (*backend.Backend, error) {
	if len(backends) <= 0 {
		return nil, NoRegisteredBackends
	}

	backendsLength := uint32(len(backends))

	counter := rr.getRouteToIndex()
	maxCount := counter + backendsLength

	for counter < maxCount {
		routeToIndex := counter % backendsLength

		if backends[routeToIndex].IsHealthy() {
			rr.setRouteToIndex(routeToIndex + 1)
			return backends[routeToIndex], nil
		}

		counter++
	}

	return nil, NoHealthyBackends
}

func (rr *RoundRobin) getRouteToIndex() uint32 {
	return rr.routeToIndex.Load()
}

func (rr *RoundRobin) setRouteToIndex(routeToIndex uint32) {
	rr.routeToIndex.Store(routeToIndex)
}
