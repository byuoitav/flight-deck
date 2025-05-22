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
	"strings"
	"time"
)

const (
	DeployURL      = "http://flood.byu.edu:2002/deploy"
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

func trimQuote(s string) string {
	if len(s) > 0 && s[0] == '\'' {
		s = s[1:]
	}

	if len(s) > 0 && s[len(s)-1] == '\'' {
		s = s[:len(s)-1]
	}
	return s
}

func errorParser(str string) string {
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		texts := strings.Split(string(line), ":")
		for i, text := range texts {
			if strings.Contains(text, "msg") {
				next := texts[i+1]
				log.Printf("MSG String: %s\n", next)
				pt := strings.Split(next, ",")
				ft := pt[0]
				clean := strings.Trim(ft, " ")
				final := trimQuote(clean)
				log.Printf("MSG Final: %s\n", final)
				return final
			}
			break
		}
	}
	return ""
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
			var final string
			logFile, err := os.ReadFile("/tmp/error.log")
			if err != nil {
				fmt.Errorf("Error Reading File: %s", err.Error())
			}
			//eep := errorParser(string(logFile))
			lines := strings.Split(string(logFile), "\n")
			for _, line := range lines {
				texts := strings.Split(string(line), ":")
				for i, text := range texts {
					if strings.Contains(text, "msg") {
						next := texts[i+1]
						log.Printf("MSG String: %s\n", next)
						pt := strings.Split(next, ",")
						ft := pt[0]
						clean := strings.Trim(ft, " ")
						final = trimQuote(clean)
						log.Printf("MSG Final: %s\n", final)
					}
					break
				}
			}
			return fmt.Errorf("Failed to deploy: %s", final)
			break
		}
		//log.Printf("No Error.log detected, Waiting 10 seconds")
		time.Sleep(5 * time.Second)

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
