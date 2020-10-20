package main

import (
	"net/http"

	"github.com/byuoitav/auth/middleware"
	"github.com/byuoitav/auth/wso2"
	"github.com/byuoitav/common"
	"github.com/byuoitav/common/log"
	"github.com/byuoitav/flight-deck/handlers"
	"github.com/labstack/echo"
)

func main() {
	port := ":8008"
	router := common.NewRouter()

	router.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "Ready for takeoff!")
	})

	client := wso2.New("", "", "https://api.byu.edu", "")

	secure := router.Group(
		"",
		echo.WrapMiddleware(client.JWTValidationMiddleware()),
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				if middleware.Authenticated(c.Request()) {
					next(c)
					return nil
				}
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized"})
			}
		},
	)

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
