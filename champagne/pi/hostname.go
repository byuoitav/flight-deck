package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/sparrc/go-ping"
)

const (
	Domain = ".byu.edu"
)

func setHostname(hn string, ignoreSubnet bool, useDHCP bool) error {
	log.Printf("Setting hostname to %s (ignoreSubnet: %v, useDHCP: %v)", hn, ignoreSubnet, useDHCP)
	ip := &net.IPNet{}

	if !useDHCP {
		// dns lookup new hostname
		var dnsError *net.DNSError

		addrs, err := net.LookupHost(hn + Domain)
		switch {
		case errors.As(err, &dnsError) && dnsError.IsNotFound:
			return ErrNotInDNS
		case err != nil:
			return fmt.Errorf("unable to lookup host: %w", err)
		case len(addrs) == 0:
			return ErrNotInDNS
		}

		// find the best IP to use

		for _, addr := range addrs {
			i := net.ParseIP(addr)
			if i != nil && !i.IsLoopback() && i.To4() != nil {
				ip.IP = i
				break
			}
		}

		if ip.IP == nil {
			return ErrNotInDNS // even though it is, there must not be an ipv4 address for it
		}

		log.Printf("Address found for %s%s in DNS: %s", hn, Domain, ip.IP.String())

		// data was locked in parent function
		data.AssignedIP = ip.IP.String()
	}

	// try pinging that IP
	var pinger *ping.Pinger
	var err error

	if ip.IP.IsUnspecified() {
		pinger, err = ping.NewPinger(ip.IP.String())
	} else {
		pinger, err = ping.NewPinger(hn + Domain)
	}

	if err != nil {
		return fmt.Errorf("unable to build pinger: %s", err)
	}

	log.Printf("Pinging %s...", pinger.Addr())
	pinger.SetPrivileged(true)
	pinger.Timeout = 5 * time.Second
	pinger.Count = 3
	pinger.Run()

	stats := pinger.Statistics()
	log.Printf("Received %v ping responses from %s (total loss: %v%%)", stats.PacketsRecv, pinger.Addr(), stats.PacketLoss)

	if stats.PacketsRecv > 0 {
		return ErrHostnameExists
	}

	// check that the ip i found works for one of the subnets i'm on
	if !useDHCP {
		ips, err := getIPs()
		if err != nil {
			return fmt.Errorf("unable to get ips: %s", err)
		}

		for _, i := range ips {
			if i.IP.Mask(i.Mask).Equal(ip.IP.Mask(i.Mask)) {
				ip.Mask = i.Mask
				break
			}
		}

		if len(ip.Mask) == 0 {
			if !ignoreSubnet {
				return ErrInvalidSubnet
			}

			// default to a /24
			ip.Mask = net.IPv4Mask(255, 255, 255, 0)
		}

		log.Printf("Using IP: %s", ip.String())
	} else {
		log.Printf("Using DHCP; Not assigning a static address")
	}

	// change the hostname
	log.Printf("Writing hostname...")
	if err = changeHostname(hn); err != nil {
		log.Printf("\n")
		return fmt.Errorf("failed to change hostname: %w", err)
	}
	log.Printf("done.")

	// change the ip
	if !useDHCP {
		log.Printf("Writing static ip...")
		if err = changeIP(ip); err != nil {
			log.Printf("\n")
			return fmt.Errorf("failed to change the ip: %w", err)
		}
		log.Printf("done.")
	}

	return nil
}

func incrementIP(ip net.IP) net.IP {
	newIP := make([]byte, len(ip))
	copy(newIP, ip)

	for i := len(newIP) - 1; i >= 0; i-- {
		newIP[i]++

		// only add to the next byte if we overflowed
		if newIP[i] != 0 {
			break
		}
	}

	return newIP
}
