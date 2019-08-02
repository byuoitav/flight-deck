package handlers

import (
	"net/http"

	"github.com/byuoitav/flight-deck/helpers"
	"github.com/labstack/echo"
)

//NewPI .
func NewPI(ctx echo.Context) error {
	// generate new id for pi
	id, err := helpers.GenerateRandomString(8)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, nil)
	}

	ret := map[string]interface{}{
		"id": id,
	}

	return ctx.JSON(http.StatusInternalServerError, ret)
}
