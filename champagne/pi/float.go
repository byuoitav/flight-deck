package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	FloatFormat     = "http://sandbag.byu.edu:10000/float"
	EnvironmentFile = "/etc/environment"
)

var (
	ErrFloatFailed = errors.New("failed to float")
)

func float() error {
	url := fmt.Printf(FloatTemplate, id)

	req, err := http.NewRequestWithContext(context.TODO, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFloatFailed, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFloatFailed, err)
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFloatFailed, err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// wait for /tmp/deployment to show up

		// get new env vars
		return source(EnvironmentFile)
	default:
		return fmt.Errorf("%w: unkown status code %d", ErrFloatFailed, resp.StatusCode)
	}
}

func saltDeployment() error {
}
