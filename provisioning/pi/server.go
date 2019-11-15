package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/sparrc/go-ping"
)

const (
	HostnameFile = "/etc/hostname"
	DHCPFile     = "/etc/dhcpcd.conf"
	ResolveFile  = "/etc/resolv.conf"

	Domain = ".byu.edu"
)

type RouteData struct {
	// auto updating
	IPs            map[string]string
	ActualHostname string

	// set by user
	DesiredHostname string
	AssignedIP      string

	// flags
	IgnoreSubnet bool
	UseDHCP      bool

	Error error

	sync.Mutex
}

type Template struct {
	templates *template.Template
}

var (
	ErrNotInDNS       = errors.New("hostname not found in DNS (qip)")
	ErrHostnameExists = errors.New("hostname is already on the network")
	ErrInvalidSubnet  = errors.New("given ip doesn't match current subnet")

	data RouteData
)

func main() {
	// check that we are root
	if os.Getuid() != 0 {
		fmt.Printf("must be run as root\n")
		os.Exit(1)
	}

	// load templates
	t := &Template{
		templates: template.Must(template.ParseGlob("./templates/*.html")),
	}

	e := echo.New()
	e.Renderer = t

	e.Static("/static", "public")

	e.GET("/pages/*", func(c echo.Context) error {
		pageName := path.Base(c.Request().URL.Path)

		data.Lock()
		defer data.Unlock()

		// reset data on start page
		if pageName == "start" {
			data.DesiredHostname = ""
			data.AssignedIP = ""
			data.Error = nil
			data.UseDHCP = false
			data.IgnoreSubnet = false
		}

		err := c.Render(http.StatusOK, pageName+".html", data)
		if err != nil {
			fmt.Printf("error rendering template %s: %v", pageName, err)
		}

		return err
	})

	setHostnameHandler := func(c echo.Context) error {
		data.Lock()
		defer data.Unlock()

		err := setHostname(data.DesiredHostname, data.IgnoreSubnet, data.UseDHCP)
		switch {
		case errors.Is(err, ErrNotInDNS):
			fmt.Printf("redirecting to 'not in dns' page\n\n")
			return c.Redirect(http.StatusTemporaryRedirect, "/pages/useDHCP")
		case errors.Is(err, ErrHostnameExists):
			fmt.Printf("redirecting to 'hostname already exists' page\n\n")
			return c.Redirect(http.StatusTemporaryRedirect, "/pages/hostnameTaken")
		case errors.Is(err, ErrInvalidSubnet):
			fmt.Printf("redirecting to 'invalid subnet' page\n\n")
			return c.Redirect(http.StatusTemporaryRedirect, "/pages/wrongSubnet")
		case err != nil:
			data.Error = err

			fmt.Printf("redirecting to 'error' page with error: %s\n\n", data.Error)
			return c.Redirect(http.StatusTemporaryRedirect, "/pages/error")
		}

		// redirect to success page
		return c.Redirect(http.StatusTemporaryRedirect, "/pages/hostnameSetSuccess")
	}

	e.GET("/ignoreSubnet", func(c echo.Context) error {
		if len(data.DesiredHostname) > 0 {
			data.Lock()
			data.IgnoreSubnet = true
			data.Unlock()

			return setHostnameHandler(c)
		}

		return c.Redirect(http.StatusTemporaryRedirect, "/pages/start")
	})

	e.GET("/useDHCP", func(c echo.Context) error {
		if len(data.DesiredHostname) > 0 {
			data.Lock()
			data.UseDHCP = true
			data.Unlock()

			return setHostnameHandler(c)
		}

		return c.Redirect(http.StatusTemporaryRedirect, "/pages/start")
	})

	// catch empty hostname
	e.GET("/hostname/", func(c echo.Context) error {
		data.Lock()
		data.Error = errors.New("Invalid hostname. Must be in the format ITB-1101-CP1")
		data.Unlock()

		return c.Redirect(http.StatusTemporaryRedirect, "/pages/error")
	})

	e.GET("/hostname/:hostname", func(c echo.Context) error {
		hn := c.Param("hostname")
		if len(hn) == 0 {
			data.Lock()
			data.Error = errors.New("Invalid hostname. Must be in the format ITB-1101-CP1")
			data.Unlock()

			return c.Redirect(http.StatusTemporaryRedirect, "/pages/error")
		}

		data.Lock()
		data.DesiredHostname = hn
		data.Unlock()

		return setHostnameHandler(c)
	})

	e.GET("/redirect", func(c echo.Context) error {
		hostname, err := os.Hostname()
		if err != nil {
			data.Lock()
			data.Error = err
			data.Unlock()
			return c.Redirect(http.StatusTemporaryRedirect, "/pages/error")
		}
		if hostname == "raspberrypi" {
			return c.Redirect(http.StatusTemporaryRedirect, "/pages/start")
		}
		return c.Redirect(http.StatusTemporaryRedirect, "/pages/floating")
	})

	// update current data every 10 seconds
	go func() {
		updateIPs := func() {
			data.Lock()
			defer data.Unlock()

			ips, err := getIPs()
			if err != nil {
				fmt.Printf("failed to get current ips: %s\n", err)
				return
			}

			// wipe out the old ip map
			data.IPs = make(map[string]string)

			for k, v := range ips {
				data.IPs[k] = v.String()
			}
		}

		updateHostname := func() {
			data.Lock()
			defer data.Unlock()

			hn, err := os.Hostname()
			if err != nil {
				fmt.Printf("failed to get current hostname: %s\n", err)
				return
			}

			data.ActualHostname = hn
		}

		// get initial data
		updateIPs()
		updateHostname()

		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			updateIPs()
			updateHostname()
		}
	}()

	if err := e.Start(":80"); err != nil {
		fmt.Printf("failed to start server: %s\n", err)
		os.Exit(1)
	}
}

// Render meets the echo templating requirement
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
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

func setHostname(hn string, ignoreSubnet bool, useDHCP bool) error {
	fmt.Printf("Setting hostname to %s (ignoreSubnet: %v, useDHCP: %v)\n", hn, ignoreSubnet, useDHCP)

	// dns lookup new hostname
	addrs, err := net.LookupHost(hn + Domain)
	if err != nil {
		return fmt.Errorf("unable to lookup host: %w", err)
	}

	switch {
	case !useDHCP && len(addrs) == 0:
		return ErrNotInDNS
	}

	// find the best IP to use
	ip := &net.IPNet{}

	for _, addr := range addrs {
		i := net.ParseIP(addr)
		if i != nil && !i.IsLoopback() && i.To4() != nil {
			ip.IP = i
			break
		}
	}

	if ip.IP == nil && !useDHCP {
		return ErrNotInDNS // even though it is, there must not be an ipv4 address for it
	}

	if !useDHCP {
		fmt.Printf("Address found for %s%s in DNS: %s\n", hn, Domain, ip.IP.String())
	}

	// try pinging that IP
	var pinger *ping.Pinger
	if ip.IP != nil {
		pinger, err = ping.NewPinger(ip.IP.String())
	} else {
		pinger, err = ping.NewPinger(hn + Domain)
	}
	if err != nil {
		return fmt.Errorf("unable to build pinger: %s", err)
	}

	fmt.Printf("Pinging %s...\n", pinger.Addr())

	pinger.Timeout = 5 * time.Second
	pinger.Count = 3
	pinger.Run()

	stats := pinger.Statistics()
	fmt.Printf("Received %v ping responses from %s (total loss: %v%%)\n", stats.PacketsRecv, pinger.Addr(), stats.PacketLoss)

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

		fmt.Printf("Using IP: %s\n", ip.String())
	} else {
		fmt.Printf("Using DHCP; Not assigning a static address\n")
	}

	// change the hostname
	fmt.Printf("Writing hostname...")
	if err = changeHostname(hn); err != nil {
		fmt.Printf("\n")
		return fmt.Errorf("failed to change hostname: %w", err)
	}
	fmt.Printf("done.\n")

	// change the ip
	if !useDHCP {
		fmt.Printf("Writing static ip...")
		if err = changeIP(ip); err != nil {
			fmt.Printf("\n")
			return fmt.Errorf("failed to change the ip: %w", err)
		}
		fmt.Printf("done.\n")
	}

	return nil
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

	return nil
}

func changeIP(ip *net.IPNet) error {
	// copy to backup

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
