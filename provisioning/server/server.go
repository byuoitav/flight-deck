package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/byuoitav/auth/wso2"
	"github.com/byuoitav/common"
	"github.com/byuoitav/common/log"
	"github.com/labstack/echo"
	"github.com/spf13/pflag"
)

var errDeviceNotFound = errors.New("Unable to find specified device in the database")

type Server struct {
	wso2Client wso2.Client
}

func main() {

	log.SetLevel("debug")

	port := pflag.IntP("port", "p", 8000, "The port that the server should run on")
	gwURL := pflag.String("gateway-url", "", "The base URL for the API gateway where flight-deck resides")
	cID := pflag.String("client-id", "", "The Client ID for the server")
	cSecret := pflag.String("client-secret", "", "The Client secret for the server")

	pflag.Parse()

	if *gwURL == "" || *cID == "" || *cSecret == "" {
		log.L.Errorf("--gateway-url, --client-id, and --client-secret must all be set")
		os.Exit(1)
	}

	server := Server{
		wso2Client: wso2.Client{
			GatewayURL:   *gwURL,
			ClientID:     *cID,
			ClientSecret: *cSecret,
		},
	}

	router := common.NewRouter()

	router.POST("/float", server.float)

	addr := fmt.Sprintf(":%d", *port)
	router.Start(addr)
}

func (s *Server) float(c echo.Context) error {

	log.L.Debugf("Starting to attempt float for ip: %s", c.RealIP())

	// Do a reverse dns lookup on the incoming ip address
	names, err := net.LookupAddr(c.RealIP())
	if err != nil {
		message := fmt.Sprintf("Error while resolving hostname for ip: %s", c.RealIP())
		log.L.Errorf("%s: %s", message, err)
		return echo.NewHTTPError(http.StatusInternalServerError, message)
	}

	log.L.Debugf("Got back names: %v", names)

	name := ""

	// Find a properly formatted name
	for _, n := range names {
		// Remove trailing domains, get only the hostname not FQD
		n = strings.SplitN(n, ".", 2)[0]
		// See if it matches ABC-123-AB2 format
		matched, _ := regexp.MatchString("^[[:alnum:]]+-[[:alnum:]]+-[[:alnum:]]+$", n)
		if matched {
			name = n
			break
		}
	}

	// If we didn't find a valid name
	if name == "" {
		message := fmt.Sprintf("No acceptable matching hostname found for ip address: %s", c.RealIP())
		log.L.Infof(message)
		return echo.NewHTTPError(http.StatusForbidden, message)
	}

	log.L.Debugf("Final name: %s", name)
	log.L.Debugf("Trying to float %s from prod", name)

	// Try floating to prod
	err = s.floatDevice(name, "prd")
	if err != nil && errors.Is(err, errDeviceNotFound) {
		// If not found in prd then try stg
		log.L.Debugf("Device %s not found in prd, trying stg", name)
		err = s.floatDevice(name, "stg")
	}

	// If there is still an error (stg returned an error too or not a not found error)
	if err != nil {
		log.L.Debugf("Got error: %s", err)
		if errors.Is(err, errDeviceNotFound) {
			msg := fmt.Sprintf("Device %s not found in prd or stg float-ship", name)
			log.L.Infof(msg)
			return echo.NewHTTPError(
				http.StatusNotFound,
				msg,
			)
		}

		return echo.NewHTTPError(
			http.StatusInternalServerError,
			err.Error(),
		)
	}

	res := struct {
		Message string
	}{
		Message: fmt.Sprintf("Successfully Floated %s", name),
	}

	c.JSON(http.StatusOK, res)

	return nil
}

func (s *Server) floatDevice(name, env string) error {

	res, err := s.wso2Client.Get(fmt.Sprintf(
		"https://api.byu.edu/domains/av/flight-deck/%s/webhook_device/%s",
		env,
		name,
	))
	if err != nil {
		return fmt.Errorf("Error while making request to flight-deck: %w", err)
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("Unable to read error body from flight-deck: %w", err)
		}

		if strings.Contains(string(body), "failed to find device") {
			return errDeviceNotFound
		}

		return fmt.Errorf("Got unknown failure from flight-deck: %s", body)
	}

	return nil
}
