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

# Make `startx` result in starting the i3 window manager
curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/xinitrc > /home/pi/.xinitrc
chmod +x /home/pi/.xinitrc

#Download the changeroom script
curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/changeroom.sh > /home/pi/changeroom.sh
chmod +x /home/pi/changeroom.sh

# Configure i3
mkdir /home/pi/.i3
curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/i3_config > /home/pi/.i3/config

# Make X start on login
curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/bash_profile > /home/pi/.bash_profile

if [ -f "$started" ]; then
	echo "Removing first boot file."
	sudo rm $bootfile
fi

sudo sh -c "reboot"
