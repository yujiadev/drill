package main

import (
	"os"
	"fmt"

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

	client := transport.NewTransportClient(
		cfg.Client.Host,
		cfg.Client.Port,
		cfg.Server.Host,
		cfg.Server.Port,	
	)

	client.Run()
}