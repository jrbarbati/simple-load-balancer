package backend

import (
	"errors"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	Url        *url.URL
	healthUri  string
	healthy    *atomic.Bool
	httpClient *http.Client
	mutex      sync.Mutex
}

var UrlParseError = errors.New("invalid url")
var ErrInvalidScheme = errors.New("missing or invalid scheme")
var ErrMissingHost = errors.New("missing host")
var ErrMissingHealthUri = errors.New("missing health uri")

func NewFromString(rawUrl string, healthUri string, httpClient *http.Client) (*Backend, error) {
	backendUrl, err := url.Parse(rawUrl)

	if err != nil {
		return nil, UrlParseError
	}

	if backendUrl.Scheme != "http" && backendUrl.Scheme != "https" {
		return nil, ErrInvalidScheme
	}

	if backendUrl.Host == "" {
		return nil, ErrMissingHost
	}

	if healthUri == "" {
		return nil, ErrMissingHealthUri
	}

	return NewFromUrl(backendUrl, healthUri, httpClient)
}

func NewFromUrl(url *url.URL, healthUri string, httpClient *http.Client) (*Backend, error) {
	healthy := &atomic.Bool{}
	healthy.Store(true)

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Backend{url, healthUri, healthy, httpClient, sync.Mutex{}}, nil
}

func (be *Backend) StartHealthCheck(cooldown time.Duration) {
	ticker := time.NewTicker(cooldown)
	defer ticker.Stop()

	for range ticker.C {
		be.CheckHealth()
	}
}

func (be *Backend) CheckHealth() {
	if !be.mutex.TryLock() {
		return
	}
	defer be.mutex.Unlock()

	healthUrl := be.Url.JoinPath(be.healthUri)

	resp, err := be.httpClient.Get(healthUrl.String())

	if err != nil {
		be.SetHealth(false)
		return
	}

	defer resp.Body.Close()

	be.SetHealth(http.StatusOK <= resp.StatusCode && resp.StatusCode < 300)
}

func (be *Backend) IsHealthy() bool {
	return be.healthy.Load()
}

func (be *Backend) SetHealth(healthy bool) {
	be.healthy.Store(healthy)
}
