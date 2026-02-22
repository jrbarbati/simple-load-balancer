package api

import (
	"context"
	"errors"
	"fmt"
	"load-balancer/internal/backend"
	"load-balancer/internal/balancer"
	"load-balancer/internal/config"
	"log"
	"net/http"
	"os/signal"
	"syscall"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server.startHealthChecks(ctx)
	httpServer := server.listenAndServe()

	<-ctx.Done()

	shutDownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return httpServer.Shutdown(shutDownCtx)
}

func (server *Server) listenAndServe() *http.Server {
	httpServer := &http.Server{Addr: fmt.Sprintf(":%d", server.port), Handler: nil}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
		}
	}()

	return httpServer
}

func (server *Server) startHealthChecks(ctx context.Context) {
	for _, lb := range server.loadBalancers {
		lb.StartHealthChecks(ctx)
	}
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

		lbs[app.Host] = balancer.New(backends, healthCheckCooldown)
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
