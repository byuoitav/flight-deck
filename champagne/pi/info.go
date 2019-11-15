package main

import (
	"fmt"
	"net"
	"strings"
)

func getIPs() (map[string]*net.IPNet, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface list: %s", err)
	}

	ips := make(map[string]*net.IPNet)

	for _, iface := range ifaces {
		// skip the docker interface
		if strings.Contains(iface.Name, "docker") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return ips, fmt.Errorf("failed to get interface list: %s", err)
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if v.IP.To4() != nil && !v.IP.IsLoopback() {
					ips[iface.Name] = v
				}
			}
		}
	}

	return ips, nil
}
