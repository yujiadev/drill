package main

import (
	"os"
	"fmt"
	//"sync"

	"drill/pkg/config"
	"drill/pkg/transport"
)

func main() {
	fmt.Println("Running as drill developing client and server")

	if len(os.Args) != 2 {
		panic("Error: Missing config file path!\ndrill <path-to-config>\n")
	}

	config_file_path := os.Args[1]	
	cfg := config.ReadThenParseServerConfig(config_file_path)

	fmt.Println(cfg)

	transport.ParseFrame([]byte{})
}
