package api

import (
	"fmt"
	"load-balancer/internal/backend"
	"load-balancer/internal/balancer"
	"load-balancer/internal/config"
	"net/http"
	"time"
)

type Server struct {
	loadBalancers map[string]*balancer.LoadBalancer
	port          int
}

func NewServerDefaultPort(pathToConfig string) (*Server, error) {
	return NewServer(0, pathToConfig)
}

func NewServer(port int, pathToConfig string) (*Server, error) {
	lbs, err := buildLoadBalancers(pathToConfig)

	if err != nil {
		return nil, err
	}

	if port == 0 {
		port = 8080
	}

	return &Server{lbs, port}, nil
}

func (server *Server) Start() error {
	http.HandleFunc("/", server.handleProxy)
	http.HandleFunc("/api/v1/loadBalancers/report", server.handleReport)

	return http.ListenAndServe(fmt.Sprintf(":%d", server.port), nil)
}

func buildLoadBalancers(pathToConfig string) (map[string]*balancer.LoadBalancer, error) {
	lbConfig, loadConfigError := config.LoadConfig(pathToConfig)
	lbs := map[string]*balancer.LoadBalancer{}

	if loadConfigError != nil {
		return nil, loadConfigError
	}

	for _, app := range lbConfig.Apps {
		duration, parseTimeoutError := time.ParseDuration(app.Timeout)
		healthCheckCooldown, parseCooldownError := time.ParseDuration(app.HealthCheckCooldown)

		if parseTimeoutError != nil {
			return nil, parseTimeoutError
		}

		if parseCooldownError != nil {
			return nil, parseCooldownError
		}

		httpClient := &http.Client{Timeout: duration}

		backends, err := buildBackends(app, httpClient)

		if err != nil {
			return nil, err
		}

		lbs[app.Host] = balancer.New(backends)
		lbs[app.Host].StartHealthChecks(healthCheckCooldown)
	}

	return lbs, nil
}

func buildBackends(app *config.ApplicationConfig, httpClient *http.Client) ([]*backend.Backend, error) {

	var backends []*backend.Backend

	for _, instance := range app.Instances {
		be, newBackendErr := backend.NewFromString(instance.Url, app.HealthUri, httpClient)

		if newBackendErr != nil {
			return nil, newBackendErr
		}

		backends = append(backends, be)
	}

	return backends, nil
}
