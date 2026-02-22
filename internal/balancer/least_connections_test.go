package balancer

import (
	"load-balancer/internal/backend"
	"testing"
)

func TestLeastConnections_NextBackend(t *testing.T) {
	scenarios := []struct {
		name                string
		backends            []*backend.Backend
		expectedConnections int32
		expectedError       error
	}{
		{
			"No Backends",
			[]*backend.Backend{},
			0,
			NoRegisteredBackends,
		},
		{
			"No Healthy Backends",
			[]*backend.Backend{
				backendWithConnections(0, false),
				backendWithConnections(0, false),
			},
			0,
			NoHealthyBackends,
		},
		{
			"Some Healthy Backends",
			[]*backend.Backend{
				backendWithConnections(12, true),
				backendWithConnections(0, false),
				backendWithConnections(5, true),
				backendWithConnections(68, true),
				backendWithConnections(2, false),
			},
			5,
			nil,
		},
		{
			"Some Healthy Backends - Tie",
			[]*backend.Backend{
				backendWithConnections(12, true),
				backendWithConnections(0, false),
				backendWithConnections(5, true),
				backendWithConnections(5, true),
				backendWithConnections(2, false),
			},
			5,
			nil,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			lc := NewLeastConnections()

			nextBackend, err := lc.NextBackend(scenario.backends)

			if scenario.expectedError != nil && err == nil {
				t.Errorf("Expected error %v, got nil", scenario.expectedError)
			}

			if scenario.expectedError == nil && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if scenario.expectedError == nil && err == nil && scenario.expectedConnections != nextBackend.ActiveConnections() {
				t.Errorf("Expected Backend with %v connections, got one with %v connections", scenario.expectedConnections, nextBackend.ActiveConnections())
			}
		})
	}
}

func BenchmarkLeastConnections_NextBackend(b *testing.B) {
	lc := NewLeastConnections()

	backends := []*backend.Backend{
		backendWithConnections(0, false),
		backendWithConnections(10, true),
		backendWithConnections(12, true),
		backendWithConnections(23, true),
		backendWithConnections(8, true),
		backendWithConnections(9, true),
		backendWithConnections(15, true),
		backendWithConnections(17, true),
		backendWithConnections(12, true),
		backendWithConnections(3, true),
		backendWithConnections(8, true),
		backendWithConnections(1, true),
		backendWithConnections(0, false),
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, _ = lc.NextBackend(backends)
	}
}

func BenchmarkParallelLeastConnections_NextBackend(b *testing.B) {
	lc := NewLeastConnections()

	backends := []*backend.Backend{
		backendWithConnections(0, false),
		backendWithConnections(10, true),
		backendWithConnections(12, true),
		backendWithConnections(23, true),
		backendWithConnections(8, true),
		backendWithConnections(9, true),
		backendWithConnections(15, true),
		backendWithConnections(17, true),
		backendWithConnections(12, true),
		backendWithConnections(3, true),
		backendWithConnections(8, true),
		backendWithConnections(1, true),
		backendWithConnections(0, false),
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = lc.NextBackend(backends)
		}
	})
}

func backendWithConnections(connections int32, healthy bool) *backend.Backend {
	be, _ := backend.NewFromString("http://test.com", "/health", nil)

	be.SetHealth(healthy)

	for range connections {
		be.AddConnection()
	}

	return be
}
