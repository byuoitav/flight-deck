package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/byuoitav/auth/wso2"
	"github.com/byuoitav/common"
	"github.com/byuoitav/common/log"
	"github.com/spf13/pflag"
)

var errDeviceNotFound = errors.New("Unable to find specified device in the database")

// Server represents the configuration necessary for the server to run
type Server struct {
	wso2Client wso2.Client
}

func main() {

	// Setup the configuration flags
	port := pflag.IntP("port", "p", 8000, "The port that the server should run on")
	gwURL := pflag.String("gateway-url", "", "The base URL for the API gateway where flight-deck resides")
	cID := pflag.String("client-id", "", "The Client ID for the server")
	cSecret := pflag.String("client-secret", "", "The Client secret for the server")

	pflag.Parse()

	// Warn if not all the required configuration flags are set
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

	router.POST("/float", server.handleFloat)

	addr := fmt.Sprintf(":%d", *port)
	router.Start(addr)
}
