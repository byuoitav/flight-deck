#!/bin/bash

printf "\n\nHi From Danny\n\n"
first="/byu/firstboot"

if [ -f "$first" ]; then
	echo "First boot."

	until $(curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/first-boot.sh > /tmp/first-boot.sh); do
		echo "Unable to download first-boot.sh - Trying again in 5 seconds."
        sleep 5
	done
	chmod +x /tmp/first-boot.sh

	/tmp/first-boot.sh
else
	echo "Second boot."

	until $(curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/second-boot.sh > /tmp/second-boot.sh); do
		echo "Unable to download second-boot.sh - Trying again in 5 seconds."
        sleep 5
	done
	chmod +x /tmp/second-boot.sh

	/tmp/second-boot.sh
fi

clear
printf "\n\n\n\n\n\n"
printf "Setup complete! I'll never see you again."
printf "\n\tPlease wait for me to reboot.\n"
sleep 10
printf "\n\nBye lol"
sleep 3
