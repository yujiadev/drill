package config

import (
	"fmt"
	"log"
	"os"

	"github.com/goccy/go-yaml"
)

type DrillConfig struct {
	Client struct {
		Enabled  bool   `yaml:"enabled"`
		Host     string `yaml:"host"`  
		Port     uint16 `yaml:"port"`
		Protocal string `yaml:"protocal"`
		Pkey     string `yaml:"pkey"`
	}
	Server struct {
		Enabled  bool   `yaml:"enabled"`
		Host     string `yaml:"host"`  
		Port     uint16 `yaml:"port"`
		Protocal string `yaml:"protocal"`
		Pkey     string `yaml:"pkey"`
	}
}

func ReadThenParseConfig(cfg_path string) DrillConfig {
	fmt.Println("ReadThenParseConfig starts reading and parsing config file")

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("ReadThenParseConfig error (query pwd): %v", err)
	}

	fmt.Println("ReadThenParseConfig is at", pwd)

	data, err := os.ReadFile(cfg_path)
	if err != nil {
		log.Fatalf("ReadThenParseConfig error (read file): %v. At %s", err, pwd)
	}

	var cfg DrillConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	fmt.Println("ReadThenParseConfig done.")
	return cfg
}