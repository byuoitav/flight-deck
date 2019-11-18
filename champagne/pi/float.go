package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	FloatURL         = "http://sandbag.byu.edu:10000/float"
	EnvironmentFile  = "/etc/environment"
	DeploymentFile   = "/tmp/deployment.log"
	SaltMinionFile   = "/etc/salt/minion"
	SaltMinionIDFile = "/etc/salt/minion_id"
)

var (
	ErrFloatFailed = errors.New("failed to float")
)

func float() error {
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, FloatURL, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFloatFailed, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFloatFailed, err)
	}
	defer resp.Body.Close()

	// idk if i need the body
	//buf, err := ioutil.ReadAll(resp.Body)
	//if err != nil {
	//	return fmt.Errorf("%w: %w", ErrFloatFailed, err)
	//}

	switch resp.StatusCode {
	case http.StatusOK:
		// wait for /tmp/deployment to show up
		count := 0
		for {
			time.Sleep(1000 * time.Second)
			count++

			if _, err := os.Stat(DeploymentFile); os.IsNotExist(err) {
				return fmt.Errorf("deployment file never showed up")
			}

			if count > 30 {
				// deployment must have failed
			}
		}

		// get new env vars
		return source(EnvironmentFile)
	default:
		return fmt.Errorf("%w: unkown status code %d", ErrFloatFailed, resp.StatusCode)
	}
}

func saltDeployment() error {
	//minionFile, err := os.Create(SaltMinionFile)
	//if err != nil {
	//	return fmt.Errorf("faield to create minion file: %w", err)
	//}

	// minionFile.Write()

	return nil
}
