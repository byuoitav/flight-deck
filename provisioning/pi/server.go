package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/labstack/echo"
	"github.com/sparrc/go-ping"
)

type RouteData struct {
	IPs      []string
	Hostname string
}

type IPInfo struct {
	Interface net.Interface
	IP        net.IP
}

type Template struct {
	templates *template.Template
}

var (
	ErrNotInDNS       = errors.New("hostname not found in DNS (qip)")
	ErrHostnameExists = errors.New("hostname is already on the network")
	ErrInvalidSubnet  = errors.New("given ip doesn't match current subnet")
)

func main() {
	// check that we are root
	if os.Getuid() != 0 {
		fmt.Printf("must be run as root\n")
		os.Exit(1)
	}

	// load templates
	t := &Template{
		// templates: template.Must(template.ParseGlob("./templates/*.html")),
	}

	e := echo.New()
	e.Renderer = t

	e.PUT("/hostname/:hostname", func(c echo.Context) error {
		hn := c.Param("hostname")
		if len(hn) == 0 {
			return c.String(http.StatusBadRequest, "must include a valid hostname")
		}

		err := setHostname(hn)
		switch {
		case errors.Is(err, ErrNotInDNS):
			// redirect to please put in qip
		case err != nil:
			return c.String(http.StatusInternalServerError, err.Error())
		}
		return c.String(http.StatusOK, "success!")
	})

	// launch chomium
	go func() {
		// wait for the server to start
		time.Sleep(5 * time.Second)

		/*
			if err := openURL("http://localhost/"); err != nil {
				fmt.Printf("failed to open browser: %s\n", err)
				os.Exit(1)
			}
		*/
	}()

	if err := e.Start(":80"); err != nil {
		fmt.Printf("failed to start server: %s\n", err)
		os.Exit(1)
	}
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func openURL(url string) error {
	// do i need to do chromium specifically
	cmd := exec.Command("xdg-open", url)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to open browser: %s", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("unable to wait for process: %s", err)
	}

	return nil
}

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

func setHostname(hn string) error {
	// dns lookup new hostname
	addrs, err := net.LookupHost(hn)
	if err != nil {
		return fmt.Errorf("unable to lookup host: %w", err)
	}

	if len(addrs) == 0 {
		return ErrNotInDNS
	}

	// find the best IP to use
	var ip net.IPNet

	for _, addr := range addrs {
		i := net.ParseIP(addr)
		if i != nil && !i.IsLoopback() && i.To4() != nil {
			ip.IP = i
			break
		}
	}

	if ip.IP == nil {
		// TODO some page for this case?
		return errors.New("no suitable ip address found")
	}

	// try pinging that IP
	pinger, err := ping.NewPinger(ip.IP.String())
	if err != nil {
		return fmt.Errorf("unable to build pinger: %s", err)
	}

	pinger.Timeout = 5 * time.Second
	pinger.Count = 3
	pinger.Run()

	stats := pinger.Statistics()
	if stats.PacketsRecv > 0 {
		return ErrHostnameExists
	}

	// check that the ip i found works for one of the subnets i'm on
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
		return ErrInvalidSubnet
	}

	return nil
}
