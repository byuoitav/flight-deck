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

type Template struct {
	templates *template.Template
}

var (
	ErrNotInDNS       = errors.New("hostname not found in DNS (qip)")
	ErrHostnameExists = errors.New("hostname is already on the network")
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

	/*
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

			ret := struct {
				Hostname string `json:"hostname"`
			}{
				Hostname: hn,
			}

			return c.JSON(http.StatusOK, ret)
		})
	*/

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
		default:
			return c.String(http.StatusOK, "success!")
		}
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
	// dns lookup new hostname
	addrs, err := net.LookupHost(hn)
	if err != nil {
		return fmt.Errorf("unable to lookup host: %w", err)
	}

	if len(addrs) == 0 {
		return ErrNotInDNS
	}

	// find the best IP to use
	var ip net.IP

	for _, addr := range addrs {
		i := net.ParseIP(addr)
		if i != nil && !i.IsLoopback() && i.To4() != nil {
			ip = i
			break
		}
	}

	if ip == nil {
		return errors.New("no suitable ip address found")
	}

	// try pinging that IP
	pinger, err := ping.NewPinger(ip.String())
	if err != nil {
		return fmt.Errorf("unable to build pinger: %s", err)
	}

	pinger.Count = 5
	pinger.Run()

	stats := pinger.Statistics()
	if stats.PacketsRecv > 0 {
		return ErrHostnameExists
	}

	return nil
}
