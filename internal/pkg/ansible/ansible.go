package ansible

import (
	"context"
	"fmt"
	"os/exec"
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
		"--vault-password-file", c.PathVaultPassword,
		"--limit", id,
	}

	cmd := exec.CommandContext(ctx, _ansibleCommand, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to run command: %w", err)
	}

	return nil
}

func (c *Client) Refloat(ctx context.Context, id string) error {
	args := []string{
		c.PathRefloatPlaybook,
		"--inventory", c.PathInventory,
		"--vault-password-file", c.PathVaultPassword,
		"--limit", id,
	}

	cmd := exec.CommandContext(ctx, _ansibleCommand, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to run command: %w", err)
	}

	return nil
}

func (c *Client) Rebuild(ctx context.Context, id string) error {
	args := []string{
		c.PathRebuildPlaybook,
		"--inventory", c.PathInventory,
		"--vault-password-file", c.PathVaultPassword,
		"--limit", id,
	}

	cmd := exec.CommandContext(ctx, _ansibleCommand, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to run command: %w", err)
	}

	return nil
}
