package handlers

import (
	"context"
	"net/http"

	//"sync"
	"time"

	"github.com/byuoitav/flight-deck/internal/app/flightdeck"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Deployer flightdeck.Deployer
}

func (h *Handlers) DeployByDeviceID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Minute)
	defer cancel()

	if err := h.Deployer.Deploy(ctx, c.Param("deviceID")); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

}

func (h *Handlers) RefloatByDeviceID(c *gin.Context) {

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Minute)
	defer cancel()

	if err := h.Deployer.Refloat(ctx, c.Param("deviceID")); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

}

func (h *Handlers) RebuildByDeviceID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Minute)
	defer cancel()

	if err := h.Deployer.Rebuild(ctx, c.Param("deviceID")); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

}
