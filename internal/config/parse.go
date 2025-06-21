package config

import (
	"log"
	"net"	
	"os"
	"encoding/base64"

	// Third party YAML builder and parser	
	"github.com/goccy/go-yaml"
)

func LoadClientYaml(cfgPath string) ReadyClientConfig {
	data := readConfigFile(cfgPath)

	var rawCfg RawClientConfig
	parseYAML(data, &rawCfg)

	return ReadyClientConfig {
		// Local	
		resolveTCPAddr(rawCfg.Client.Addr),

		// Remote
		resolveUDPAddr(rawCfg.Server.Addr),
		rawCfg.Server.Protocol,
		base64ToBytes(rawCfg.Server.Pkey),
	}
}

func LoadServerYaml(cfgPath string) ReadyServerConfig {
	data := readConfigFile(cfgPath)

	var rawCfg RawServerConfig
	parseYAML(data, &rawCfg)

	return ReadyServerConfig {
		resolveUDPAddr(rawCfg.Server.Addr),
		rawCfg.Server.Protocol,	
		base64ToBytes(rawCfg.Server.Pkey),
	}
}

func readConfigFile(path string) []byte {
	data, err := os.ReadFile(path)

	if err != nil {
		log.Panicf("Error reading configuration file on %s. %s\n", path, err)
	}

	return data	
}

func parseYAML(data []byte, cfg any) {
	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Panicf("Error parsing YAML. %s\n", err)	
	}
}

func base64ToBytes(str string) []byte {
	key, err := base64.StdEncoding.DecodeString(str)

	if err != nil {
		log.Panicf("Error decoding base64 string %s. %s\n", str, err)
	}

	return key
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