package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/labstack/echo"
)

func main() {
	e := echo.New()

	e.GET("/ips", func(c echo.Context) error {
		ips, err := getIPs()
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, ips)
	})

	e.GET("/hostname", func(c echo.Context) error {
		hn, err := os.Hostname()
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		return c.String(http.StatusOK, hn)
	})

	e.PUT("/hostname/:hostname", func(c echo.Context) error {
		return nil
	})

	if err := e.Start(":8080"); err != nil {
		fmt.Printf("failed to start server: %s\n", err)
		os.Exit(1)
	}
}

func openURL(url string) error {
	cmd := exec.Command("xdg-open", url)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("unable to open browser: %s", err)
	}
}

func getIPs() ([]string, error) {
	var ips []string

	ifaces, err := net.Interfaces()
	if err != nil {
		return ips, fmt.Errorf("failed to get interface list: %s", err)
	}

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
				ipv4 := v.IP.To4()

				if ipv4 != nil && !v.IP.IsLoopback() {
					ips = append(ips, ipv4.String())
				}
			}
		}
	}

	return ips, nil
}

func setHostname(hn string) error {
	return nil
}
