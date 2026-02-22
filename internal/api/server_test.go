package api

import (
	"load-balancer/internal/balancer"
	"reflect"
	"testing"
)

func TestDetermineStrategy(t *testing.T) {
	scenarios := []struct {
		input    string
		expected balancer.Strategy
	}{
		{"", &balancer.RoundRobin{}},
		{"round_robin", &balancer.RoundRobin{}},
		{"least_connections", &balancer.LeastConnections{}},
		{"invalid input", &balancer.RoundRobin{}},
	}

	for _, scenario := range scenarios {
		strategy := determineStrategy(scenario.input)

		if reflect.TypeOf(strategy) != reflect.TypeOf(scenario.expected) {
			t.Errorf("determineStrategy(%v) expected %T, got %T", scenario.input, scenario.expected, strategy)
		}
	}
}
