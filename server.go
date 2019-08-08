package main

import (
	"net/http"

	"github.com/byuoitav/authmiddleware"
	"github.com/byuoitav/common"
	"github.com/byuoitav/common/log"
	"github.com/byuoitav/flight-deck/handlers"
	"github.com/labstack/echo"
)

func main() {
	port := ":8008"
	router := common.NewRouter()

	// unautheticated routes
	router.Static("/*", "public")

	secure := router.Group("", echo.WrapMiddleware(authmiddleware.Authenticate))

	/* secure routes */
	secure.GET("/webhook_device/:hostname", handlers.DeployByHostname)

	err := router.StartServer(&http.Server{
		Addr:           port,
		MaxHeaderBytes: 1024 * 10,
	})
	if err != nil {
		log.L.Errorf("failed to start http server: %v", err)
	}
}
