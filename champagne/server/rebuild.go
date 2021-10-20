package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/byuoitav/common/log"
	"github.com/labstack/echo"
)

// handleFloat handles the incoming request to float a pi
func (s *Server) handleRebuild(c echo.Context) error {

	log.L.Debugf("Starting to attempt a rebuild for ip: %s", c.RealIP())

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
		// Remove trailing domains, get only the hostname not FQDN
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

	// If there is still an error (still not found or other error)
	if err != nil {
		log.L.Debugf("Got error: %s", err)
		if errors.Is(err, errDeviceNotFound) {
			msg := fmt.Sprintf("Device %s not found in any flight-deck environment", name)
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

	// Worked successfully, return success
	res := struct {
		Message string
	}{
		Message: fmt.Sprintf("Successfully Rebuilt %s", name),
	}

	c.JSON(http.StatusOK, res)

	return nil
}

// floatDevice attempts to get flight-deck to float to the given name from the
// given location
func (s *Server) rebuildDevice(name, env string) error {

	// Make the request to flight-deck
	res, err := s.wso2Client.Get(fmt.Sprintf(
		"https://api.byu.edu/domains/av/flight-deck/%s/rebuild/%s",
		env,
		name,
	))
	if err != nil {
		return fmt.Errorf("Error while making request to flight-deck: %w", err)
	}

	// If we got an error back try to figure out what went wrong
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("Unable to read error body from flight-deck: %w", err)
		}

		// If the device isn't in flight-deck's database then return errDeviceNotFound
		if strings.Contains(string(body), "failed to get device") {
			return errDeviceNotFound
		}

		return fmt.Errorf("Got unknown failure from flight-deck: %s", body)
	}

	return nil
}
