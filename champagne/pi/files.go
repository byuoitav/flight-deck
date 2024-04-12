package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-ini/ini"
)

const (
	HostnameFile = "/etc/hostname"
	DHCPFile     = "/etc/dhcpcd.conf"
	HostsFile    = "/etc/hosts"
	ReleaseFile  = "/etc/os-release"
)

func ReadOSReleaseInfo(configfile string) map[string]string {
	cfg, err := ini.Load(configfile)
	if err != nil {
		fmt.Errorf("Fail to read file: ", err)
	}

	ConfigParams := make(map[string]string)
	ConfigParams["ID"] = cfg.Section("").Key("ID").String()
	ConfigParams["VERSION_CODENAME"] = cfg.Section("").Key("VERSION_CODENAME").String()
	ConfigParams["VERSION_ID"] = cfg.Section("").Key("VERSION_ID").String()

	return ConfigParams
}

func waitForFile(ctx context.Context, name string, checkForContent bool) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats, err := os.Stat(name)
			if err != nil {
				continue // file doesn't exist yet
			}

			if !checkForContent || stats.Size() > 0 {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

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

	toWrite := fmt.Sprintf("\n127.0.0.1\t%s", hn)
	n, err = hostsFile.WriteString(toWrite)
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

	// Check if OS is Bookworm or an older version
	// Starting with bookworm, debian/raspbian started using network manager
	// We are only deploying Bookworm from here on out but we want a way to fall back for a little bit
	var err error

	OSReleaseInfo := ReadOSReleaseInfo(ReleaseFile)
	OSRelease := OSReleaseInfo["VERSION_ID"]
	OSReleaseFloat, err := strconv.ParseFloat(OSRelease, 64)
	if err != nil {
		fmt.Errorf("Failed to convert OS Release version from string to float: %v\n", err.Error())
	}

	OSReleaseInt := int(OSReleaseFloat)

	log.Printf("Setting up static IP address\n")

	if OSReleaseInt >= 12 {
		log.Printf("Setting up static IP address\n")

		router := ip.IP.Mask(ip.Mask)
		router = incrementIP(router)

		perm := "sudo"
		app := "nmcli"
		conn := "connection"
		mod := "modify"
		old_profile := "Wired connection 1"
		new_profile := "byu-av"
		connid := "connection.id"
		ipv4addName := "ipv4.address"
		ipv4add := ip.String()
		ipvgatewayName := "ipv4.gateway"
		ipv4gateway := router.String()
		ipv4methodName := "ipv4.method"
		ipv4method := "manual"
		ipv4dnsName := "ipv4.dns"
		ipv4dns := "127.0.0.1,10.8.0.19,10.8.0.26"

		nmcmd := exec.Command(app, conn, mod, old_profile, connid, new_profile)
		log.Printf("Command: %s\n", nmcmd.String())

		cmd := exec.Command(perm, app, conn, mod, new_profile, ipv4addName, ipv4add, ipvgatewayName, ipv4gateway, ipv4methodName, ipv4method, ipv4dnsName, ipv4dns)
		log.Printf("Command: %s\n", cmd.String())

		var out bytes.Buffer
		var stderr bytes.Buffer
		nmcmd.Stdout = &out
		nmcmd.Stderr = &stderr

		err = nmcmd.Run()
		if err != nil {
			log.Printf(fmt.Sprintf(err.Error()) + ":" + stderr.String())
			log.Printf("Failed to run command: %v\n", err.Error())
			return fmt.Errorf("Failed to run command: %v", err.Error())
		}
		log.Printf("Output: %v\n", out.String())

		cmd.Stdout = &out
		cmd.Stderr = &stderr

		err = cmd.Run()
		if err != nil {
			log.Printf(fmt.Sprintf(err.Error()) + ":" + stderr.String())
			log.Printf("Failed to run command: %v\n", err.Error())
			return fmt.Errorf("Failed to run command: %v", err.Error())
		}
		log.Printf("Output: %v\n", out.String())
	}

	if OSReleaseInt <= 11 {
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
		str.WriteString("\ninterface eth0\n")
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
	}

	return nil

}
