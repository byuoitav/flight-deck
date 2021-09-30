package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/byuoitav/auth/middleware"
	"github.com/byuoitav/auth/wso2"
	"github.com/byuoitav/flight-deck/internal/app/flightdeck/handlers"
	//"github.com/byuoitav/flight-deck/internal/app/flightdeck/opa"
	"github.com/byuoitav/flight-deck/internal/pkg/ansible"
	"github.com/gin-gonic/gin"
	"github.com/gwatts/gin-adapter"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func MiddlewareWso2() gin.HandlerFunc {

	//client := wso2.New("", "", "https://api.byu.edu", "")
	//authentic := client.JWTValidationMiddleware()
	fmt.Printf("Getting JWT and testing\n")

	return func(c *gin.Context) {

		fmt.Printf("Request: %s\n", c.Request)
		if middleware.Authenticated(c.Request) {
			fmt.Printf("Middleware Authentication Successful\n")
			c.Next()
			return
		}
		fmt.Printf("WSO2 Authentication Failed\n")
		fmt.Printf("Output of JWT: %s\n", c.Request)
		c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized"})
	}
}

func main() {
	var (
		port     int
		logLevel string
		wg       sync.WaitGroup
		//opaURL              string
		//opaToken            string
		pathDeployPlaybook  string
		pathRefloatPlaybook string
		pathRebuildPlaybook string
		pathInventory       string
		pathVaultPassword   string
	)

	pflag.CommandLine.IntVarP(&port, "port", "P", 8080, "port to run the server on")
	pflag.StringVarP(&logLevel, "log-level", "L", "", "level to log at. refer to https://godoc.org/go.uber.org/zap/zapcore#Level for options")
	//pflag.StringVarP(&opaURL, "opa-address", "a", "", "OPA Address (Full URL)")
	//pflag.StringVarP(&opaToken, "opa-token", "t", "", "OPA Token")
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

	r := gin.New()
	r.Use(gin.Recovery())

	client := wso2.New("", "", "https://api.byu.edu", "")

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, fmt.Sprintf("Flightdeck Standing By..."))
		return
	})
	/*
		o := opa.Client{
			URL:   opaURL,
			Token: opaToken,
		}
	*/
	// WSO2 and OPA Middleware added to /api/v1
	api := r.Group("/api/v1/")
	api.Use(adapter.Wrap(client.JWTValidationMiddleware()))
	api.Use(MiddlewareWso2())
	//api.Use(o.Authorize())

	api.POST("/deploy/:deviceID", handlers.DeployByDeviceID)
	api.POST("/refloat/:deviceID", func(c *gin.Context) {
		cCp := c.Copy()
		wg.Add(1)
		go func() {
			handlers.RefloatByDeviceID(cCp)
		}()
		//c.JSON(http.StatusOK, fmt.Sprintf("Refloat Command Sent"))
		defer wg.Wait()
	})

	api.POST("/rebuild/:deviceID", handlers.RebuildByDeviceID)

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
