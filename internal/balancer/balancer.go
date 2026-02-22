package balancer

import (
	"context"
	"errors"
	"load-balancer/internal/backend"
	"sync/atomic"
	"time"
)

const (
	RoundRobinStrategy       = "round_robin"
	LeastConnectionsStrategy = "least_connections"
)

var NoRegisteredBackends = errors.New("no registered backends")
var NoHealthyBackends = errors.New("no healthy backends available")

type Strategy interface {
	NextBackend([]*backend.Backend) (*backend.Backend, error)
}

type LoadBalancer struct {
	backends            []*backend.Backend
	strategy            Strategy
	healthCheckCooldown time.Duration
}

func New(backends []*backend.Backend, strategy Strategy, healthCheckCooldown time.Duration) *LoadBalancer {
	routeToIndex := &atomic.Uint64{}
	routeToIndex.Store(0)

	return &LoadBalancer{backends, strategy, healthCheckCooldown}
}

func (lb *LoadBalancer) StartHealthChecks(ctx context.Context) {
	for _, be := range lb.backends {
		go be.StartHealthCheck(ctx, lb.healthCheckCooldown)
	}
}

func (lb *LoadBalancer) GetNextBackend() (*backend.Backend, error) {
	return lb.strategy.NextBackend(lb.backends)
}

func (lb *LoadBalancer) GetBackends() []*backend.Backend {
	return lb.backends
}
