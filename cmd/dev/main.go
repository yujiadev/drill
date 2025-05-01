package main

import (
	"os"
	"fmt"

	"drill/pkg/config"
)

func main() {
	if len(os.Args) != 2 {
		panic("Error: Missing config file path!\ndrill <path-to-config>\n")
	}

	config_file_path := os.Args[1]

	config.ReadThenParseConfig(config_file_path)

	fmt.Println("Running as drill developing client and server")
}