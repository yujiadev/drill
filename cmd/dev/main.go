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

	proxy := transport.NewHttpsProxy(
		cfg.Client.Host,
		cfg.Client.Port,
		cfg.Server.Host,
		cfg.Server.Port,
		cfg.Client.Pkey,	
		cfg.Server.Protocal,
	)

	server := transport.NewServer(
		cfg.Server.Host,
		cfg.Server.Port,
		cfg.Server.Pkey,
		cfg.Server.Protocal,
	)

	var wg sync.WaitGroup	

	wg.Add(2)
	go ProxyExecute(proxy, &wg)
	go ServerExecute(server, &wg)
	wg.Wait()
}

func ProxyExecute(proxy transport.HttpsProxy, wg *sync.WaitGroup) {
	proxy.Run()
	wg.Done()
}

func ServerExecute(server transport.Server, wg *sync.WaitGroup) {
	server.Run()
	wg.Done()
}