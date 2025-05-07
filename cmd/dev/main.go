package main

import (
	"os"
	"fmt"
	"sync"

	"drill/pkg/config"
	"drill/pkg/transport"
)

func main() {
	fmt.Println("Running as drill developing client and server")

	if len(os.Args) != 2 {
		panic("Error: Missing config file path!\ndrill <path-to-config>\n")
	}

	config_file_path := os.Args[1]
	cfg := config.ReadThenParseConfig(config_file_path)

	clientTransport := transport.NewClientTransport(
		cfg.Client.Host,
		cfg.Client.Port,
		cfg.Server.Host,
		cfg.Server.Port,
		0,
		cfg.Server.Pkey,
		cfg.Server.Protocal,
	)

	serverTransport := transport.NewServerTransport(
		cfg.Server.Host,
		cfg.Server.Port,
		cfg.Server.Pkey,
		cfg.Server.Protocal,	
	)

	var wg sync.WaitGroup	
	wg.Add(1)
	go clientExecute(clientTransport)
	go serverExecute(serverTransport)
	wg.Wait()
}

func clientExecute(txp transport.ClientTransport) {
	txp.Run()
}

func serverExecute(txp transport.ServerTransport) {
	txp.Run()
}