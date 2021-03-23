package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func main() {
	var (
		port     int
		logLevel string
	)

	pflag.CommandLine.IntVarP(&port, "port", "P", 8080, "port to run the server on")
	pflag.StringVarP(&logLevel, "log-level", "L", "", "level to log at. refer to https://godoc.org/go.uber.org/zap/zapcore#Level for options")

	pflag.Parse()

	// ctx for setup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, log := logger(logLevel)
	defer log.Sync() // nolint:errcheck

	// TODO wso2

	r := gin.New()
	r.Use(gin.Recovery())

	api := r.Group("/api/v1/")
	api.GET("/deployToDevice/:deviceID", func(c *gin.Context) {
	})

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal("unable to bind listener", zap.Error(err))
	}

	log.Info("Starting server", zap.String("on", lis.Addr().String()))
	err = r.RunListener(lis)
	switch {
	case errors.Is(err, http.ErrServerClosed):
	case err != nil:
		log.Fatal("failed to server", zap.Error(err))
	}
}
