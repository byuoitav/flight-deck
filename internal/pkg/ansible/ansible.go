package ansible

import (
	"context"
	"fmt"

	//"os/exec"

	"github.com/codeskyblue/go-sh"
)

const (
	_ansibleCommand = "ansible-playbook"
)

type Client struct {
	PathDeployPlaybook  string
	PathRefloatPlaybook string
	PathRebuildPlaybook string
	PathInventory       string
	PathVaultPassword   string
}

func (c *Client) Deploy(ctx context.Context, id string) error {
	args := []string{
		c.PathDeployPlaybook,
		"--inventory", c.PathInventory,
		"--limit", id,
		"--vault-password-file", c.PathVaultPassword,
	}

	session := sh.NewSession()
	session.SetDir("/")
	cmd := session.Command(_ansibleCommand, args).Run()
	session.ShowCMD = true
	fmt.Printf("Command: %s\n", cmd)
	/*
		cmd := exec.CommandContext(ctx, _ansibleCommand, args...)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("unable to run command: %w", err)
		}

		cmd.Wait()
	*/
	return nil
}

func (c *Client) Refloat(ctx context.Context, id string) error {
	args := []string{
		c.PathRefloatPlaybook,
		"--inventory", c.PathInventory,
		"--limit", id,
		"--vault-password-file", c.PathVaultPassword,
	}

	session := sh.NewSession()
	session.SetDir("/")
	cmd := session.Command(_ansibleCommand, args).Run()
	session.ShowCMD = true

	//cmd := exec.CommandContext(ctx, _ansibleCommand, args...)
	fmt.Printf("Command: %s\n", cmd)
	/*
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("unable to run command: %w", err)
		}

		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("unable to finish execution: %w", err)
		}
	*/
	return nil
}

func (c *Client) Rebuild(ctx context.Context, id string) error {
	args := []string{
		c.PathRebuildPlaybook,
		"--inventory", c.PathInventory,
		"--limit", id,
		"--vault-password-file", c.PathVaultPassword,
	}

	session := sh.NewSession()
	session.SetDir("/")
	cmd := session.Command(_ansibleCommand, args).Run()
	session.ShowCMD = true
	fmt.Printf("Command: %s\n", cmd)
	/*
		cmd := exec.CommandContext(ctx, _ansibleCommand, args...)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("unable to run command: %w", err)
		}
	*/
	return nil
}
