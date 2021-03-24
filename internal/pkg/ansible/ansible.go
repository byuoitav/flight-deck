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
	PathDeployPlaybook string
	PathInventory      string
	PathVaultPassword  string
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
