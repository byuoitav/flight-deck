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
	DeployURL      = "http://ansible.av.byu.edu:8080/api/v1/deploy/"
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

	FullDeployURL := DeployURL + hostname

	data.Lock()
	data.ProgressMessage = "Deploying from Ansible..."
	data.ProgressPercent = 30

	log.Printf("%s", data.ProgressMessage)
	data.Unlock()

	// Creating new Post Request to start ansible deployment
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, FullDeployURL, nil)
	if err != nil {
		return fmt.Errorf("failed to deploy: %w", err)
	}

	log.Printf("Making GET request to %s", FullDeployURL)

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
