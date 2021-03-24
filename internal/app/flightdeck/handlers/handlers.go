package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/byuoitav/flight-deck/internal/app/flightdeck"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Deployer flightdeck.Deployer
}

func (h *Handlers) DeployByDeviceID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	if err := h.Deployer.Deploy(ctx, c.Param("deviceID")); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusOK)
}
