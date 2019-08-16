#!/bin/bash
# This script should live in /byu/ on the rasbian img. It will run once on the first boot of the pi, and then disable the service.

sleep 15
printf "\n\nHi From Danny\n\n"
sudo chvt 2

first="/byu/firstboot"

if [ -f "$first" ]; then
	echo "First boot."

	# download pi-setup script
	until $(curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/first-boot.sh > /tmp/first-boot.sh); do
		echo "Unable to download first-boot.sh - Trying again in 5 seconds."
        sleep 5
	done
	chmod +x /tmp/first-boot.sh

	/tmp/first-boot.sh

	echo "Removing first boot file and rebooting"
    rm $first
    sleep 3

    sudo reboot
else
	sleep 30
	sudo chvt 2

	echo "Second boot."

	# download second-boot script
	until $(curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/second-boot.sh > /tmp/second-boot.sh); do
		echo "Trying again."
	done
	chmod +x /tmp/second-boot.sh

	/tmp/second-boot.sh

	echo "Removing symlink to startup script."
	sudo systemctl disable first-boot.service
    sleep 3
fi

clear
printf "\n\n\n\n\n\n"
printf "Setup complete! I'll never see you again."
printf "\n\tPlease wait for me to reboot.\n"
sleep 10
printf "\n\nBye lol"
sleep 3
