package main

import (
	"os"
	"fmt"

	"drill/pkg/config"
	//"drill/pkg/x_crypto"
	"drill/pkg/proxy"
)

func main() {
	fmt.Println("Running as drill developing client and server")

	if len(os.Args) != 2 {
		panic("Error: Missing config file path!\ndrill <path-to-config>\n")
	}

	config_file_path := os.Args[1]
	cfg := config.ReadThenParseConfig(config_file_path)

	https_proxy := proxy.New(cfg.Client.Host, cfg.Client.Port)

	https_proxy.Run()
}