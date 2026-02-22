package balancer

import (
	"context"
	"errors"
	"load-balancer/internal/backend"
	"sync/atomic"
	"time"
)

var NoRegisteredBackends = errors.New("no registered backends")
var NoHealthyBackends = errors.New("no healthy backends available")

type LoadBalancer struct {
	backends            []*backend.Backend
	routeToIndex        *atomic.Uint64
	backendsCount       uint64
	healthCheckCooldown time.Duration
}

func New(backends []*backend.Backend, healthCheckCooldown time.Duration) *LoadBalancer {
	routeToIndex := &atomic.Uint64{}
	routeToIndex.Store(0)

	return &LoadBalancer{backends, routeToIndex, uint64(len(backends)), healthCheckCooldown}
}

func (lb *LoadBalancer) StartHealthChecks(ctx context.Context) {
	for _, be := range lb.backends {
		go be.StartHealthCheck(ctx, lb.healthCheckCooldown)
	}
}

func (lb *LoadBalancer) GetNextBackend() (*backend.Backend, error) {
	if len(lb.backends) <= 0 {
		return nil, NoRegisteredBackends
	}

	counter := lb.getRouteToIndex()
	maxCount := counter + uint64(len(lb.backends))

	for counter < maxCount {
		routeToIndex := counter % lb.backendsCount

		if lb.backends[routeToIndex].IsHealthy() {
			lb.setRouteToIndex(routeToIndex + 1)
			return lb.backends[routeToIndex], nil
		}

		counter++
	}

	return nil, NoHealthyBackends
}

func (lb *LoadBalancer) getRouteToIndex() uint64 {
	return lb.routeToIndex.Load()
}

func (lb *LoadBalancer) setRouteToIndex(routeToIndex uint64) {
	lb.routeToIndex.Store(routeToIndex)
}

func (lb *LoadBalancer) GetBackends() []*backend.Backend {
	return lb.backends
}
