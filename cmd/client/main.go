package main

import (
	"os"
	"fmt"

	"drill/pkg/config"
	txp "drill/pkg/transport"
)

func main() {
	fmt.Println("Running as drill developing client and server")

	if len(os.Args) != 2 {
		panic("Error: Missing config file path!\ndrill <path-to-config>\n")
	}

	config_file_path := os.Args[1]	
	cfg := config.ReadThenParseClientConfig(config_file_path)


	clientTxp := txp.NewClientTransport(
		cfg.Client.Address,
		cfg.Server.Address,
		cfg.Server.Pkey,
		cfg.Server.Protocol,
	)

	clientTxp.Run()
}
