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

	cipher := x_crypto.New(config.Client.Pkey)

	msg := []byte("hello world!")
	ciphertext := cipher.Encrypt(&msg)
	plaintext, err := cipher.Decrypt(&ciphertext)

	if err != nil {

	}

	fmt.Println(msg)
	fmt.Println(plaintext)

	fmt.Println("Running as drill developing client and server")
}