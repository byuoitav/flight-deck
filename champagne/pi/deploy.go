package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

const (
	DeployURL      = "http://10.5.34.222:2001/deploy"
	DeploymentFile = "/tmp/deployment.log"
)

var (
	ErrDeviceNotFound = errors.New("device not found in production or stage")

	FinalProgressMessages = []string{
		"honestly i'm not sure what it's doing but just give it a minute",
		"if you're having issues, please call 801-422-KENG",
		"dirty mike is finding the boys...",
	}
)

type dcompose struct {
	Services map[string]ms `yaml:"services,omitempty"`
}

type ms struct {
	Ports       []string           `yaml:"ports,omitempty"`
	Command     string             `yaml:"command,omitempty"`
	Environment []string           `yaml:"environment,omitempty"`
	NetworkMode string             `yaml:"network_mode,omitempty"`
	Restart     string             `yaml:"restart,omitempty"`
	TTY         string             `yaml:"tty,omitempty"`
	Logging     map[string]options `yaml:"logging,omitempty"`
	Image       string             `yaml:"image,omitempty"`
}

type options struct {
	MaxSize       string `yaml:"max-size,omitempty"`
	Mode          string `yaml:"mode,omitempty"`
	MaxBufferSize string `yaml:"max-buffer-size,omitempty"`
}

func ansible_deploy(hostname string) error {
	log.Printf("Deploying from Ansible...")

	// Removing error.log file if one already exists
	if _, err := os.Stat("/tmp/error.log"); err == nil {
		err := os.Remove("/tmp/error.log")
		if err != nil {
			fmt.Errorf("Failed to remove error file")
		}
	}

	data.Lock()
	data.ProgressMessage = "Deploying from Ansible..."
	data.ProgressPercent = 99

	log.Printf("%s", data.ProgressMessage)
	data.Unlock()

	// Creating new Post Request to start ansible deployment
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, DeployURL, nil)
	if err != nil {
		return fmt.Errorf("failed to deploy: %w", err)
	}

	log.Printf("Making the request to %s", DeployURL)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to deploy: %w", err)
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to deploy: %w", err)
	}

	log.Printf("Response %d:\n%s", resp.StatusCode, buf)

	// Check for an Error file in the temp dirctory to know that deploy failed
	for {
		if _, err := os.Stat("/tmp/error.log"); err == nil {
			_, err := os.ReadFile("/tmp/error.log")
			if err != nil {
				fmt.Errorf("Error Reading File: %s")
			}
			return fmt.Errorf("Failed to deploy: Please check error log for details")
			break
		}
		//log.Printf("No Error.log detected, Waiting 10 seconds")
		time.Sleep(10 * time.Second)

	}

	switch resp.StatusCode {
	case http.StatusOK:
		log.Printf("Waiting for deployment to finish.....")
		return nil
	case http.StatusForbidden:
		return fmt.Errorf("failed to deploy: %w", ErrDeviceNotFound)
	case http.StatusNotFound:
		return fmt.Errorf("failed to deploy: %s", buf)
	case http.StatusInternalServerError:
		return fmt.Errorf("failed to deploy: unknown error: %s", buf)
	default:
		return fmt.Errorf("failed to deploy: unknown status code %d: %s", resp.StatusCode, buf)
	}

	return nil
}

func finishDeployment() error {

	data.Lock()
	data.ProgressMessage = "finished! rebooting in 1 minute."
	data.ProgressPercent = 99
	log.Printf("%s", data.ProgressMessage)
	data.Unlock()

	// so that the ui can refresh
	time.Sleep(30 * time.Second)

	// schedule a reboot (will shutdown in 1 minute)
	cmd := exec.Command("shutdown", "-r")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "shutdown -r", err)
	}

	// disable myself (will kill the program!!!!)
	cmd = exec.Command("systemctl", "disable", "pi-setup.service")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "systemctl disable pi-setup.service", err)
	}

	return nil
}
