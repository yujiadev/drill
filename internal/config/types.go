package config

import (
	"net"
)

// The struct that matches the "client" section in the client.yaml file
type ClientConfig struct {
	Addr string 		`yaml:"address"`
	Pkey string			`yaml:"pkey"`
}

// The struct that matches the "server" section in the client.yaml 
// and server.yaml
type ServerConfig struct {
	Addr string 		`yaml:"address"`
	Protocol string		`yaml:"protocol"` 
	Pkey string			`yaml:"pkey"`
}

// The struct structurally represent the client.yaml
type RawClientConfig struct {
	Client ClientConfig
	Server ServerConfig
}

// The struct structurally represents the server.yaml
type RawServerConfig struct {
	Server ServerConfig
}

// Ready to use client side config
type ReadyClientConfig struct {
	// Local-related configurations
	LocalAddr *net.TCPAddr 

	// Remote-related configurations
	RemoteAddr 		*net.UDPAddr	
	RemoteProtocol  string
	RemotePkey      []byte
}

// Ready to use server side config
type ReadyServerConfig struct {
	Addr      *net.UDPAddr 
	Protocol  string
	Pkey      []byte
}