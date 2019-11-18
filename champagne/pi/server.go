package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/labstack/echo"
)

const (
	HostnameFile = "/etc/hostname"
	DHCPFile     = "/etc/dhcpcd.conf"

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

	// progress
	ProgressTitle   string
	ProgressMessage string
	ProgressPercent int // 1 - 100

	Error error

	sync.Mutex
}

type Template struct {
	templates *template.Template
}

// Render meets the echo templating requirement
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

var (
	ErrNotInDNS       = errors.New("hostname not found in DNS (qip)")
	ErrHostnameExists = errors.New("hostname is already on the network")
	ErrInvalidSubnet  = errors.New("given ip doesn't match current subnet")
	ErrFloatFailed    = errors.New("failed to float")

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

	// html/css/js
	e.Static("/static", "public")
	e.GET("/pages/*", serveHTMLHandler)

	// general endpoints
	e.GET("/redirect", redirectHandler)

	// first boot stuff (set hn/ip)
	e.GET("/ignoreSubnet", ignoreSubnetHandler)
	e.GET("/useDHCP", allowDHCPHandler)
	e.GET("/hostname/", emptyHostnameHandler) // catch empty hostnames
	e.GET("/hostname/:hostname", hostnameSetHandler)

	// float/second boot stuff (set hn/ip)
	e.GET("/float", floatHandler)

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

func serveHTMLHandler(c echo.Context) error {
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
}

func redirectHandler(c echo.Context) error {
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
}

func setHostnameHandler(c echo.Context) error {
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

	// if it works, then start the update process
	go func() {
		if err = updateAndReboot(); err != nil {
			data.Lock()
			data.Error = fmt.Errorf("failed to update and reboot: %s", err)
			data.Unlock()

			fmt.Printf("failed to update and reboot: %s\n", err)
		}
	}()

	data.ProgressTitle = "Set Hostname - Updating Pi"
	data.ProgressPercent = 0

	// redirect to success page
	return c.Redirect(http.StatusTemporaryRedirect, "/pages/progress")
}

func hostnameSetHandler(c echo.Context) error {
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
}

func emptyHostnameHandler(c echo.Context) error {
	data.Lock()
	data.Error = errors.New("Invalid hostname. Must be in the format ITB-1101-CP1")
	data.Unlock()

	return c.Redirect(http.StatusTemporaryRedirect, "/pages/error")
}

func allowDHCPHandler(c echo.Context) error {
	if len(data.DesiredHostname) > 0 {
		data.Lock()
		data.UseDHCP = true
		data.Unlock()

		return setHostnameHandler(c)
	}

	return c.Redirect(http.StatusTemporaryRedirect, "/pages/start")
}

func ignoreSubnetHandler(c echo.Context) error {
	if len(data.DesiredHostname) > 0 {
		data.Lock()
		data.IgnoreSubnet = true
		data.Unlock()

		return setHostnameHandler(c)
	}

	return c.Redirect(http.StatusTemporaryRedirect, "/pages/start")
}

func floatHandler(c echo.Context) error {
	// hit the float endpoint
	if len(data.ActualHostname) == 0 || data.ActualHostname == "raspberrypi" {
		return c.Redirect(http.StatusTemporaryRedirect, "/redirect")
	}
}
