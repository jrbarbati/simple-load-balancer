package main

import (
	"load-balancer/internal/api"
	"log"
)

func main() {
	server, err := api.NewServerDefaultPort("config.yaml")

	if err != nil {
		log.Fatal(err)
	}

	log.Println("load balancer listening on :8080")
	log.Fatal(server.Start())
}
