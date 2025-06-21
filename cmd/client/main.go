package main

import (
	"log"
	"sync"

	"drill/internal/config"
	"drill/internal/transport"
)

func main() {
	log.Println("Client started")
	cfg := config.LoadClientYaml("configs/client.yaml")
	var wg sync.WaitGroup

	client := transport.NewClientTransport(
		cfg.LocalAddr,
		cfg.RemoteAddr,
		cfg.RemoteProtocol,
		cfg.RemotePkey,
		&wg,
	)

	wg.Add(1)
	go client.Run()
	wg.Wait()
}