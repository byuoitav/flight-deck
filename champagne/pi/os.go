package main

import (
	"fmt"
	"os/exec"
	"time"

	"golang.org/x/sys/unix"
)

func updateAndReboot() error {
	data.Lock()
	data.ProgressMessage = "updating apt"
	data.Unlock()

	// update apt
	if err := exec.Command("apt", "update").Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "apt update", err)
	}

	data.Lock()
	data.ProgressPercent = 15
	data.ProgressMessage = "upgrading packages"
	data.Unlock()

	// upgrade packages
	if err := exec.Command("apt", "-y", "upgrade").Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "apt -y upgrade", err)
	}

	data.Lock()
	data.ProgressPercent = 75
	data.ProgressMessage = "removing leftover packages"
	data.Unlock()

	// remove/clean leftover junk
	if err := exec.Command("apt", "-y", "autoremove").Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "apt -y autoremove", err)
	}

	data.Lock()
	data.ProgressPercent = 90
	data.ProgressMessage = "cleaning apt cache"
	data.Unlock()

	if err := exec.Command("apt", "-y", "autoclean").Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "apt -y autoclean", err)
	}

	data.Lock()
	data.ProgressPercent = 99
	data.ProgressMessage = "rebooting"
	data.Unlock()

	time.Sleep(10 * time.Second)

	// reboot the pi (for more info, look at the man page for reboot(2))
	if err := unix.Reboot(unix.LINUX_REBOOT_CMD_RESTART); err != nil {
		return fmt.Errorf("failed to reboot: %w", err)
	}

	return nil
}
