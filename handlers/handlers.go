package handlers

import (
	"net/http"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/flight-deck/helpers"
	"github.com/labstack/echo"
)

// DeployByHostname handles the echo request to deploy to a single device
func DeployByHostname(ctx echo.Context) error {
	hostname := ctx.Param("hostname")

	reports, err := helpers.DeployByHostname(hostname)
	if err != nil {
		log.L.Warnf(err.Error())
		return ctx.JSON(http.StatusInternalServerError, err.String())
	}

	return ctx.JSON(http.StatusOK, reports)
}
