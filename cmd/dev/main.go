package main

import (
	"os"
	"fmt"

	"drill/pkg/config"
	"drill/pkg/x_crypto"
)

func main() {
	if len(os.Args) != 2 {
		panic("Error: Missing config file path!\ndrill <path-to-config>\n")
	}

	config_file_path := os.Args[1]
	config := config.ReadThenParseConfig(config_file_path)


	fmt.Println(config)	

	x_crypto.Encrypt()


	fmt.Println("Running as drill developing client and server")
}