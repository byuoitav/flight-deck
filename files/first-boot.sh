#!/usr/bin/env bash

# This script is used to install and set up dependencies on a newly wiped/installed Raspberry Pi
# For clean execution, run this script inside of the /tmp directory with `./pi-setup.sh`
# The script assumes the username of the autologin user is "pi"
bootfile="/usr/local/games/firstboot"
started="/usr/local/games/setup-started"

# Run the `sudo.sh` code block to install necessary packages and commands
curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/sudo.sh > /tmp/sudo.sh
chmod +x /tmp/sudo.sh
sudo sh -c "bash /tmp/sudo.sh"

if [ -f "$started" ]; then
	echo "Removing first boot file."
	sudo rm $bootfile
fi

sudo sh -c "reboot"
