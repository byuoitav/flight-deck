package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	HostnameFile = "/etc/hostname"
	DHCPFile     = "/etc/dhcpcd.conf"
	HostsFile    = "/etc/hosts"
)

func changeHostname(hn string) error {
	f, err := os.Create(HostnameFile)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", HostnameFile, err)
	}
	defer f.Close()

	n, err := f.WriteString(hn)
	switch {
	case err != nil:
		return fmt.Errorf("failed to write: %w", err)
	case len(hn) != n:
		return fmt.Errorf("failed to write: wrote %v/%v bytes", n, len(hn))
	}

	// update /etc/hosts
	hostsFile, err := os.OpenFile(HostsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", HostsFile, err)
	}
	defer hostsFile.Close()

	toWrite := fmt.Sprintf("127.0.0.1\t%s", hn)
	n, err = f.WriteString(toWrite)
	switch {
	case err != nil:
		return fmt.Errorf("failed to write: %w", err)
	case len(toWrite) != n:
		return fmt.Errorf("failed to write: wrote %v/%v bytes", n, len(toWrite))
	}

	return nil
}

func changeIP(ip *net.IPNet) error {
	// TODO copy to backup

	f, err := os.OpenFile(DHCPFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", DHCPFile, err)
	}
	defer f.Close()

	// TODO get this from the current subnet, just like the mask?
	// default router address is .1
	router := ip.IP.Mask(ip.Mask)
	router = incrementIP(router)

	var str strings.Builder

	// TODO interface name
	str.WriteString("interface eth0\n")
	str.WriteString(fmt.Sprintf("static ip_address=%s\n", ip.String()))
	str.WriteString(fmt.Sprintf("static routers=%s\n", router.String()))
	str.WriteString("static domain_name_servers=127.0.0.1 10.8.0.19 10.8.0.26\n")

	toWrite := str.String()
	n, err := f.WriteString(toWrite)
	switch {
	case err != nil:
		return fmt.Errorf("failed to write: %w", err)
	case len(toWrite) != n:
		return fmt.Errorf("failed to write: wrote %v/%v bytes", n, len(toWrite))
	}

	// update resolve.conf (to use dnsmasq)
	/*
		resolveFile, err := os.Create(ResolveFile)
		if err != nil {
			return fmt.Errorf("failed to open file %q: %w", ResolveFile, err)
		}
		defer resolveFile.Close()

		resolveWrite := []byte("nameserver    127.0.0.1")
		n, err = resolveFile.Write(resolveWrite)
		switch {
		case err != nil:
			return fmt.Errorf("failed to write: %w", err)
		case len(resolveWrite) != n:
			return fmt.Errorf("failed to write: wrote %v/%v bytes", n, len(resolveWrite))
		}
	*/

	return nil
}
