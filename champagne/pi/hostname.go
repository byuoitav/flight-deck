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

	var dnsError *net.DNSError
	ip := &net.IPNet{}

	pinger, err := ping.NewPinger(hn + Domain)
	switch {
	case errors.As(err, &dnsError) && dnsError.IsNotFound:
		if !useDHCP {
			return ErrNotInDNS
		}
	case err != nil:
		return fmt.Errorf("unable to build pinger: %s", err)
	}

	if pinger != nil {
		// data was locked in parent function
		ip.IP = pinger.IPAddr().IP
		data.AssignedIP = ip.IP.String()

		log.Printf("Address found for %s%s in DNS: %s", hn, Domain, ip.IP.String())
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
