package model

import (
	"net"
)

type Endpoint struct {
	Addr     string   `json:"addr"`
	Names    []string `json:"names"`
	Port     int      `json:"port"`
	DataSize int      `json:"data_size"`
}

func NewEndpoint(ip net.IP, port int, names []string, size int) Endpoint {
	return Endpoint{
		Addr:     ip.String(),
		Port:     port,
		Names:    names,
		DataSize: size,
	}
}

type FlowLog struct {
	Client Endpoint `json:"client"`
	Server Endpoint `json:"server"`

	Latency  float64 `json:"latency"`
	Duration float64 `json:"duration"`
}
