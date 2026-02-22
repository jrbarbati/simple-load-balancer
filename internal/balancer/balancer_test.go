package balancer

import (
	"errors"
	"load-balancer/internal/backend"
	"strconv"
	"testing"
)

func TestLoadBalancer_GetNextBackend(t *testing.T) {
	scenarios := []struct {
		name            string
		healthStates    []bool
		expectedIndices []int
		expectedError   error
	}{
		{"No Registered Backends", []bool{}, []int{-1}, NoRegisteredBackends},
		{"No Healthy Backends", []bool{false, false, false}, []int{-1}, NoHealthyBackends},
		{"One Healthy Backend", []bool{false, true, false}, []int{1, 1, 1, 1}, nil},
		{"All Healthy Backends", []bool{true, true, true}, []int{0, 1, 2, 0}, nil},
		{"One Unhealthy Backend", []bool{false, true, true}, []int{1, 2, 1, 2}, nil},
	}

	for _, scenario := range scenarios {
		var backends []*backend.Backend

		for i, healthState := range scenario.healthStates {
			be, _ := backend.NewFromString("http://test"+strconv.Itoa(i)+".com", "/test", nil)

			be.SetHealth(healthState)

			backends = append(backends, be)
		}

		lb := New(backends)

		for i, expectedIndex := range scenario.expectedIndices {
			actualBackend, actualErr := lb.GetNextBackend()

			if scenario.expectedError != nil && actualErr != nil && !errors.Is(actualErr, scenario.expectedError) {
				t.Errorf("%v (iteration %v) GetNextBackend() error = %v, expected error = %v", scenario.name, i+1, actualErr, scenario.expectedError)
			}

			if scenario.expectedError == nil && actualErr != nil {
				t.Errorf("%v (iteration %v) GetNextBackend() returned an unexpected error = %v", scenario.name, i+1, actualErr)
			}

			if scenario.expectedError != nil && actualErr == nil {
				t.Errorf("%v (iteration %v) GetNextBackend() returned no error when it should have, expected error = %v", scenario.name, i+1, scenario.expectedError)
			}

			if scenario.expectedError == nil && actualErr == nil && !actualBackend.IsHealthy() {
				t.Errorf("%v (iteration %v) GetNextBackend() returned an unhealthy backend", scenario.name, i+1)
			}

			if scenario.expectedError == nil && actualErr == nil && actualBackend.IsHealthy() && backends[expectedIndex] != actualBackend {
				t.Errorf("%v (iteration %v) GetNextBackend() returned the wrong backend, expected %v, actual %v", scenario.name, i+1, backends[expectedIndex], actualBackend)
				break
			}

			t.Logf("%v (iteration %v) PASSED", scenario.name, i+1)
		}
	}
}

func BenchmarkGetNextBackend(b *testing.B) {
	var backends []*backend.Backend

	for i, healthState := range []bool{true, false, false, true, true, false, true} {
		be, _ := backend.NewFromString("http://test"+strconv.Itoa(i)+".com", "/test", nil)
		be.SetHealth(healthState)

		backends = append(backends, be)
	}

	lb := New(backends)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, _ = lb.GetNextBackend()
	}
}

func BenchmarkParallelGetNextBackend(b *testing.B) {
	var backends []*backend.Backend

	for i, healthState := range []bool{true, false, false, true, true, false, true} {
		be, _ := backend.NewFromString("http://test"+strconv.Itoa(i)+".com", "/test", nil)
		be.SetHealth(healthState)

		backends = append(backends, be)
	}

	lb := New(backends)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = lb.GetNextBackend()
		}
	})
}
