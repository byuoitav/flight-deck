package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/byuoitav/flight-deck/internal/app/flightdeck/handlers"
	"github.com/byuoitav/flight-deck/internal/pkg/ansible"
	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func main() {
	var (
		port     int
		logLevel string

		pathDeployPlaybook  string
		pathRefloatPlaybook string
		pathRebuildPlaybook string
		pathInventory       string
		pathVaultPassword   string
	)

	pflag.CommandLine.IntVarP(&port, "port", "P", 8080, "port to run the server on")
	pflag.StringVarP(&logLevel, "log-level", "L", "", "level to log at. refer to https://godoc.org/go.uber.org/zap/zapcore#Level for options")
	pflag.StringVarP(&pathDeployPlaybook, "deploy-playbook", "", "", "path to the ansible deployment playbook")
	pflag.StringVarP(&pathRefloatPlaybook, "refloat-playbook", "", "", "path to the ansible refloat playbook")
	pflag.StringVarP(&pathRebuildPlaybook, "rebuild-playbook", "", "", "path to the ansible rebuild playbook")
	pflag.StringVarP(&pathInventory, "inventory", "", "", "path to the ansible inventory file")
	pflag.StringVarP(&pathVaultPassword, "vault-password", "", "", "path to the ansible vault password file")

	pflag.Parse()

	// ctx for setup
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()

	_, log := logger(logLevel)
	defer log.Sync() // nolint:errcheck

	handlers := handlers.Handlers{
		Deployer: &ansible.Client{
			PathDeployPlaybook:  pathDeployPlaybook,
			PathRefloatPlaybook: pathRefloatPlaybook,
			PathRebuildPlaybook: pathRebuildPlaybook,
			PathInventory:       pathInventory,
			PathVaultPassword:   pathVaultPassword,
		},
	}

	// TODO wso2

	r := gin.New()
	r.Use(gin.Recovery())

	api := r.Group("/api/v1/")
	api.GET("/deployToDevice/:deviceID", handlers.DeployByDeviceID)

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
