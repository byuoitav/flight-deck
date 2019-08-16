#!/bin/bash

if [ "$EUID" -ne 0 ]; then
	echo "Must be run as root/sudo."
	exit 1
fi

# get binary for xssstart
curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/rpi-xssstart > /usr/local/bin/xssstart
chmod +x /usr/local/bin/xssstart

# get start script
curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/screenshutoff.sh > /usr/local/bin/screenshutoff
chmod +x /usr/local/bin/screenshutoff

echo "You're all good to go!"
