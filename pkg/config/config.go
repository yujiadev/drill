package config

import (
	"log"
	"net"
	"os"

	"github.com/goccy/go-yaml"
)

type RawClientConfig struct {
	Client struct {
		Address string `yaml:"address"`
		Pkey    string `yaml:"pkey"`
	}
	Server RawServerConfig `yaml:"server"`
}

type RawServerConfig struct {
	Address  string `yaml:"address"`
	Protocol string `yaml:"protocal"` // Note: YAML tag matches existing typo in config files
	Pkey     string `yaml:"pkey"`
}

type Client struct {
	Address *net.TCPAddr
	Pkey    string
}

type Server struct {
	Address  *net.UDPAddr
	Protocol string
	Pkey     string
}

type ClientConfig struct {
	Client Client
	Server Server
}

func ReadThenParseClientConfig(cfgPath string) ClientConfig {
	log.Println("Start reading and parsing client config file")
	data := readConfigFile(cfgPath)
	
	var rawCfg RawClientConfig
	parseYAML(data, &rawCfg)
	log.Println("Finish reading and parsing client config file.")

	return ClientConfig{
		Client: Client{
			Address: resolveTCPAddr(rawCfg.Client.Address),
			Pkey:    rawCfg.Client.Pkey,
		},
		Server: Server{
			Address:  resolveUDPAddr(rawCfg.Server.Address),
			Protocol: rawCfg.Server.Protocol,
			Pkey:     rawCfg.Server.Pkey,
		},
	}
}

func ReadThenParseServerConfig(cfgPath string) Server {
	log.Println("Start reading and parsing server config file")
	data := readConfigFile(cfgPath)

	var rawCfg RawServerConfig
	parseYAML(data, &rawCfg)
	log.Println("Finish reading and parsing server config file.")

	return Server{
		Address:  resolveUDPAddr(rawCfg.Address),
		Protocol: rawCfg.Protocol,
		Pkey:     rawCfg.Pkey,
	}
}

func readConfigFile(path string) []byte {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting working directory: %v", err)
	}
	log.Printf("Current working directory: %s", pwd)

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Error reading config file: %v (pwd: %s)", err, pwd)
	}
	return data
}

func parseYAML(data []byte, cfg interface{}) {
	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Fatalf("Error parsing YAML: %v", err)
	}
}

func resolveTCPAddr(address string) *net.TCPAddr {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		log.Fatalf("Error resolving TCP address %q: %v", address, err)
	}
	return addr
}

func resolveUDPAddr(address string) *net.UDPAddr {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		log.Fatalf("Error resolving UDP address %q: %v", address, err)
	}
	return addr
}