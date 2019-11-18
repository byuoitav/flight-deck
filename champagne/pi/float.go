package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"
)

const (
	FloatURL         = "http://sandbag.byu.edu:2001/float"
	EnvironmentFile  = "/etc/environment"
	DeploymentFile   = "/tmp/deployment.log"
	SaltMinionFile   = "/etc/salt/minion"
	SaltMinionIDFile = "/etc/salt/minion_id"
)

var (
	ErrFloatFailed = errors.New("failed to float")
)

func float() error {
	fmt.Printf("Floating...\n")

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, FloatURL, nil)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFloatFailed, err)
	}

	fmt.Printf("Making GET request to %s\n", FloatURL)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFloatFailed, err)
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFloatFailed, err)
	}

	fmt.Printf("Response:\n%s\n", buf)

	switch resp.StatusCode {
	case http.StatusOK:
		// wait for /tmp/deployment to show up
		for i := 0; i < 32; i++ {
			fmt.Printf("Looking for deployment file\n")
			time.Sleep(1 * time.Second)

			if _, err := os.Stat(DeploymentFile); err == nil {
				fmt.Printf("Got deployment file\n")
				return source(EnvironmentFile)
			}
		}

		return fmt.Errorf("deployment file never showed up")
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrFloatFailed, buf)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrFloatFailed, buf)
	case http.StatusInternalServerError:
		return fmt.Errorf("%w: unkown error: %s", ErrFloatFailed, buf)
	default:
		return fmt.Errorf("%w: unkown status code %d: %s", ErrFloatFailed, resp.StatusCode, buf)
	}
}

func saltDeployment() error {
	fmt.Printf("Starting salt deployment\n")

	data.Lock()
	data.ProgressMessage = "creating salt minion file"
	data.ProgressPercent = 10

	fmt.Printf("%s\n", data.ProgressMessage)
	data.Unlock()

	minionFile, err := os.Create(SaltMinionFile)
	if err != nil {
		return fmt.Errorf("faield to create minion file: %w", err)
	}
	defer minionFile.Close()

	data.Lock()
	data.ProgressMessage = "writing salt minion file"
	data.ProgressPercent = 15

	fmt.Printf("%s\n", data.ProgressMessage)
	data.Unlock()

	// write master address
	str := fmt.Sprintf("master: %s", os.Getenv("SALT_MASTER_HOST"))
	n, err := minionFile.WriteString(str)
	switch {
	case err != nil:
		return fmt.Errorf("unable to write to minion file: %w", err)
	case n != len(str):
		return fmt.Errorf("unable to write to minion file: wrote %v/%v bytes", n, len(str))
	}

	// write master finger
	str = fmt.Sprintf("master_finger: %s", os.Getenv("SALT_MASTER_FINGER"))
	n, err = minionFile.WriteString(str)
	switch {
	case err != nil:
		return fmt.Errorf("unable to write to minion file: %w", err)
	case n != len(str):
		return fmt.Errorf("unable to write to minion file: wrote %v/%v bytes", n, len(str))
	}

	// write startup states
	str = fmt.Sprintf("startup_states: 'highstate'")
	n, err = minionFile.WriteString(str)
	switch {
	case err != nil:
		return fmt.Errorf("unable to write to minion file: %w", err)
	case n != len(str):
		return fmt.Errorf("unable to write to minion file: wrote %v/%v bytes", n, len(str))
	}

	data.Lock()
	data.ProgressMessage = "deleting salt minion id file"
	data.ProgressPercent = 30

	fmt.Printf("%s\n", data.ProgressMessage)
	data.Unlock()

	// delete minion id file
	if err := os.Remove(SaltMinionIDFile); err != nil {
		return fmt.Errorf("failed to remove salt minion file: %w", err)
	}

	data.Lock()
	data.ProgressMessage = "restarting salt minion"
	data.ProgressPercent = 35

	fmt.Printf("%s\n", data.ProgressMessage)
	data.Unlock()

	// restart salt minion
	cmd := exec.Command("systemctl", "restart", "salt-minion")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "systemctl restart salt-minion", err)
	}

	// wait for deployment stuff to finish
	fmt.Printf("waiting for deployment to finish (5 minutes).\ncur time: %v\n", time.Now())
	for i := 0; i < 30; i++ {
		time.Sleep(10 * time.Second)
		data.Lock()

		// these are so random, but i want to make salt look like it takes longer :)
		switch {
		case i < 8:
			data.ProgressMessage = "downloading av-control-api"
		case i >= 8 && i <= 24:
			data.ProgressMessage = "downloading salt config files"
		default:
			data.ProgressMessage = "honestly i'm not sure what it's doing but just give it a minute"
		}

		data.ProgressPercent = 35 + 2*i
		data.Unlock()
	}

	// disable myself
	cmd = exec.Command("systemctl", "disable", "pi-setup.service")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "systemctl restart salt-minion", err)
	}

	data.Lock()
	data.ProgressMessage = "finished! rebooting!"
	data.ProgressPercent = 99

	fmt.Printf("%s\n", data.ProgressMessage)
	data.Unlock()

	time.Sleep(5 * time.Second)
	return reboot()
}
