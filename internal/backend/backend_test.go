package backend

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackend_NewFromString(t *testing.T) {
	scenarios := []struct {
		RawUrl        string
		HealthUri     string
		ExpectedError error
	}{
		{"http://localhost", "/health", nil},
		{"https://localhost", "/health", nil},
		{"localhost", "/health", ErrInvalidScheme},
		{"://broken", "/health", UrlParseError},
		{"", "/health", ErrInvalidScheme},
		{"http://", "/health", ErrMissingHost},
		{"http://valid.com", "/health", nil},
		{"http://valid.com", "", ErrMissingHealthUri},
	}

	for _, scenario := range scenarios {
		_, err := NewFromString(scenario.RawUrl, scenario.HealthUri, nil)

		if scenario.ExpectedError != nil && err == nil {
			t.Errorf("NewFromString(%v) should have returned an error", scenario.RawUrl)
			continue
		}

		if scenario.ExpectedError == nil && err != nil {
			t.Errorf("NewFromString(%v) should not have returned an error", scenario.RawUrl)
		}

		if scenario.ExpectedError != nil && err != nil && !errors.Is(err, scenario.ExpectedError) {
			t.Errorf("NewFromString(%v) did not return the correct error", scenario.RawUrl)
		}
	}
}

func TestBackend_CheckHealth(t *testing.T) {
	scenarios := []struct {
		name           string
		statusCode     int
		shouldErr      bool
		expectedHealth bool
	}{
		{"200 OK", 200, false, true},
		{"300 REDIRECT", 300, false, false},
		{"400 BAD REQUEST", 400, false, false},
		{"500 INTERNAL SERVER ERROR", 500, false, false},
		{"Connection Closed/Timeout", -1, true, false},
	}

	for _, scenario := range scenarios {
		scenario := scenario

		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(scenario.statusCode)
		}))

		if scenario.shouldErr {
			testServer.Close()
		}

		be, _ := NewFromString(testServer.URL, "/health", testServer.Client())

		be.CheckHealth()

		if scenario.expectedHealth != be.IsHealthy() {
			t.Errorf("%v should have returned %v", scenario.name, scenario.expectedHealth)
		}

		if !scenario.shouldErr {
			testServer.Close()
		}
	}
}

func TestBackend_CheckHealth_PreventStackingCalls(t *testing.T) {
	counter := atomic.Uint32{}
	counter.Store(0)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
		counter.Add(1)
	}))
	defer testServer.Close()

	be, _ := NewFromString(testServer.URL, "/health", testServer.Client())

	go be.CheckHealth()
	time.Sleep(10 * time.Millisecond) // Give it time to acquire the lock
	be.CheckHealth()

	time.Sleep(50 * time.Millisecond)
	be.CheckHealth()
	time.Sleep(60 * time.Millisecond)

	if counter.Load() != 2 {
		t.Errorf("CheckHealth() should have ran %v times, actual = %v times", 2, counter.Load())
	}
}

func TestBackend_SetHealth(t *testing.T) {
	scenarios := []struct {
		SetTo    bool
		Expected bool
	}{
		{SetTo: true, Expected: true},
		{SetTo: false, Expected: false},
	}

	for _, scenario := range scenarios {
		be, _ := NewFromString("http://www.test.com", "/health", nil)

		be.SetHealth(scenario.SetTo)
		actual := be.IsHealthy()

		if scenario.Expected != actual {
			t.Errorf("SetHealth(%v) expected %v, actual %v", scenario.SetTo, scenario.Expected, actual)
		}
	}
}

func BenchmarkNewFromString(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, _ = NewFromString("http://www.test.com", "/health", nil)
	}
}

func BenchmarkBackend_IsHealthy(b *testing.B) {
	be, _ := NewFromString("http://www.test.com", "/health", nil)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		be.IsHealthy()
	}
}
