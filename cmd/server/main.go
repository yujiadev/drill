package main

import (
	"log"
	"sync"

	"drill/internal/config"
	"drill/internal/transport"
)

func main() {
	log.Println("Server started")
	cfg := config.LoadServerYaml("configs/server.yaml")

	var wg sync.WaitGroup

	server := transport.NewServerTransport(
		cfg.Addr,
		cfg.Protocol,
		cfg.Pkey,
		&wg,
	)

	wg.Add(1)
	go server.Run()
	wg.Wait()
}