package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/byuoitav/auth/middleware"
	"github.com/byuoitav/auth/wso2"
	"github.com/byuoitav/flight-deck/internal/app/flightdeck/handlers"
	"github.com/byuoitav/flight-deck/internal/app/flightdeck/opa"
	"github.com/byuoitav/flight-deck/internal/pkg/ansible"
	"github.com/gin-gonic/gin"
	adapter "github.com/gwatts/gin-adapter"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func MiddlewareWso2(log *zap.Logger) gin.HandlerFunc {

	//client := wso2.New("", "", "https://api.byu.edu", "")
	//authentic := client.JWTValidationMiddleware()
	log.Debug("Getting JWT and testing")

	return func(c *gin.Context) {

		log.Debug(fmt.Sprintf("Request: %s", c.Request))
		if middleware.Authenticated(c.Request) {
			log.Info("Middleware Authentication Successful")
			c.Next()
			return
		}
		log.Info("WSO2 Authentication Failed")
		log.Debug(fmt.Sprintf("Output of JWT: %s", c.Request))
		c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized"})
	}
}

func main() {
	var (
		port                int
		logLevel            string
		opaURL              string
		opaToken            string
		pathDeployPlaybook  string
		pathRefloatPlaybook string
		pathRebuildPlaybook string
		pathInventory       string
		pathVaultPassword   string
	)

	pflag.CommandLine.IntVarP(&port, "port", "P", 8080, "port to run the server on")
	pflag.StringVarP(&logLevel, "log-level", "L", "", "level to log at. refer to https://godoc.org/go.uber.org/zap/zapcore#Level for options")
	pflag.StringVarP(&opaURL, "opa-address", "a", "", "OPA Address Full URL")
	pflag.StringVarP(&opaToken, "opa-token", "t", "", "OPA Token")
	pflag.StringVarP(&pathDeployPlaybook, "deploy-playbook", "", "", "path to the ansible deployment playbook")
	pflag.StringVarP(&pathRefloatPlaybook, "refloat-playbook", "", "", "path to the ansible refloat playbook")
	pflag.StringVarP(&pathRebuildPlaybook, "rebuild-playbook", "", "", "path to the ansible rebuild playbook")
	pflag.StringVarP(&pathInventory, "inventory", "", "", "path to the ansible inventory file")
	pflag.StringVarP(&pathVaultPassword, "vault-password", "", "", "path to the ansible vault password file")

	pflag.Parse()

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
		c.JSON(http.StatusOK, gin.H{"Success": "Flightdeck Standing By..."})
		return
	})

	o := opa.Client{
		URL:   opaURL,
		Token: opaToken,
	}

	// WSO2 and OPA Middleware added to /api/v1
	api := r.Group("/api/v1/")
	api.Use(adapter.Wrap(client.JWTValidationMiddleware()))
	api.Use(MiddlewareWso2(log))
	api.Use(o.Authorize(log))

	api.POST("/deploy/:deviceID", func(c *gin.Context) {
		cCp := c.Copy()
		go func() {
			handlers.DeployByDeviceID(cCp)
		}()
		c.JSON(http.StatusOK, gin.H{"Success": "Deployment Command Sent"})
	})

	api.POST("/refloat/:deviceID", func(c *gin.Context) {
		cCp := c.Copy()
		go func() {
			handlers.RefloatByDeviceID(cCp)
		}()
		c.JSON(http.StatusOK, gin.H{"Success": "Refloat Command Sent"})
	})

	api.POST("/rebuild/:deviceID", func(c *gin.Context) {
		cCp := c.Copy()
		go func() {
			handlers.RebuildByDeviceID(cCp)
		}()
		c.JSON(http.StatusOK, gin.H{"Success": "Rebuild Command Sent"})
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
