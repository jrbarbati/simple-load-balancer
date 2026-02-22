package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
)

type LoadBalancerReport struct {
	Apps []*LoadBalancerReportApp `json:"apps"`
}

type LoadBalancerReportApp struct {
	Host      string                        `json:"host"`
	Instances []*LoadBalancerReportInstance `json:"instances"`
}

type LoadBalancerReportInstance struct {
	Url     string `json:"url"`
	Healthy bool   `json:"healthy"`
}

func (server *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("received request: %s %s %s\n", r.Method, r.Host, r.URL)

	host, _, splitErr := net.SplitHostPort(r.Host)

	if splitErr != nil {
		host = r.Host
	}

	lb := server.loadBalancers[host]

	if lb == nil {
		http.Error(w, "No load balancer found", http.StatusInternalServerError)
		log.Printf("no load balancer found for %s", host)
		return
	}

	be, err := lb.GetNextBackend()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("GetNextBackend: %v", err)
		return
	}

	be.AddConnection()
	defer be.ReleaseConnection()

	httputil.NewSingleHostReverseProxy(be.Url).ServeHTTP(w, r)
}

func (server *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(buildLoadBalancerReport(server))

	if err != nil {
		http.Error(w, fmt.Sprintf("error encoding JSON: %v", err), http.StatusInternalServerError)
		log.Printf("Error encoding JSON: %v", err)
		return
	}
}

func buildLoadBalancerReport(server *Server) LoadBalancerReport {
	var report LoadBalancerReport
	report.Apps = make([]*LoadBalancerReportApp, len(server.loadBalancers))
	index := 0

	for host, lb := range server.loadBalancers {
		report.Apps[index] = &LoadBalancerReportApp{host, make([]*LoadBalancerReportInstance, len(lb.GetBackends()))}

		for i, instance := range lb.GetBackends() {
			report.Apps[index].Instances[i] = &LoadBalancerReportInstance{instance.Url.String(), instance.IsHealthy()}
		}

		index++
	}

	return report
}
