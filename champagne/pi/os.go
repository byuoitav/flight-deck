package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"golang.org/x/sys/unix"
)

func fixTime() error {
	fmt.Printf("Fixing time\n")

	cmd := exec.Command("ntpdate", "tick.byu.edu")
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to fix time: %s", err)
	}

	return nil
}

func updateAndReboot() error {
	data.Lock()
	data.ProgressMessage = "fixing time"
	data.ProgressPercent = 5
	data.Unlock()

	if err := fixTime(); err != nil {
		return err
	}

	data.Lock()
	data.ProgressMessage = "updating apt"
	data.ProgressPercent = 15
	data.Unlock()

	fmt.Printf("Updating apt\n")

	// update apt
	cmd := exec.Command("apt", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "apt update", err)
	}

	fmt.Printf("\nUpgrading packages\n")

	data.Lock()
	data.ProgressPercent = 30
	data.ProgressMessage = "upgrading packages"
	data.Unlock()

	// upgrade packages
	cmd = exec.Command("apt", "-y", "upgrade")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "apt -y upgrade", err)
	}

	fmt.Printf("\nRemoving leftover packages\n")

	data.Lock()
	data.ProgressPercent = 75
	data.ProgressMessage = "removing leftover packages"
	data.Unlock()

	// remove/clean leftover junk
	cmd = exec.Command("apt", "-y", "autoremove")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "apt -y autoremove", err)
	}

	fmt.Printf("\nCleaning apt cache\n")

	data.Lock()
	data.ProgressPercent = 90
	data.ProgressMessage = "cleaning apt cache"
	data.Unlock()

	cmd = exec.Command("apt", "-y", "autoclean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "apt -y autoclean", err)
	}

	fmt.Printf("\n\n\nDone! Rebooting!!\n")
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
