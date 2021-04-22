package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	//"math/rand"
	"net/http"
	"os"
	"os/exec"
	"time"
	//"github.com/docker/docker/api/types"
	//"github.com/docker/docker/client"
	//"gopkg.in/yaml.v2"
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
	data.ProgressMessage = "Deploying from Ansible...Will Take 10-15 Minutes"
	data.ProgressPercent = 30

	log.Printf("%s", data.ProgressMessage)
	data.Unlock()

	// Creating new Post Request to start ansible deployment
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, FullDeployURL, nil)
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
		// wait for deployment file and environment file
		/*
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			log.Printf("Waiting for deployment file...")

			if err := waitForFile(ctx, DeploymentFile, false); err != nil {
				return fmt.Errorf("deployment file never showed up: %s", err)
			}

			log.Printf("Waiting for environment file...")

			if err := waitForFile(ctx, EnvironmentFile, true); err != nil {
				return fmt.Errorf("environment file never showed up: %s", err)
			}

			return source(EnvironmentFile)
		*/
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

	// Finishing deployment and removing the setup service from the system and rebooting pi
	err = finishDeployment()
	if err != nil {
		return fmt.Errorf("failed to finish deployment: %w", err)
	}
	return nil
}

func finishDeployment() error {
	/*
		// get a random final message
		rand.Seed(time.Now().UnixNano())
		idx := rand.Intn(len(FinalProgressMessages))

		timeout := 600
		done := false

		for {
			time.Sleep(7 * time.Second)
			timeout += 7
			data.Lock()

			runningDockers, err := cli.ContainerList(context.TODO(), types.ContainerListOptions{})
			if err != nil {
				return fmt.Errorf("unable to get list of running docker containers: %s", err)
			}

			switch {
			case len(runningDockers) < len(dockers.Services):
				data.ProgressMessage = fmt.Sprintf("downloaded %d/%d applications", len(runningDockers), len(dockers.Services))
			case len(runningDockers) == len(dockers.Services):
				done = true
				break
			default:
				data.ProgressMessage = FinalProgressMessages[idx]
			}

			data.ProgressPercent = int(100 * float32(len(runningDockers)) / float32(len(dockers.Services)))
			data.Unlock()
			if done {
				break
			}
		}
	*/
	// schedule a reboot (will shutdown in 1 minute)
	cmd := exec.Command("shutdown", "-r")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "shutdown -r", err)
	}

	data.Lock()
	data.ProgressMessage = "finished! rebooting in 1 minute."
	data.ProgressPercent = 99
	log.Printf("%s", data.ProgressMessage)
	data.Unlock()

	// so that the ui can refresh
	time.Sleep(30 * time.Second)

	// disable myself (will kill the program!!!!)
	cmd = exec.Command("systemctl", "disable", "pi-setup.service")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "systemctl disable pi-setup.service", err)
	}

	return nil
}
