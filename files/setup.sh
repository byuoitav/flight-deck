#!/bin/bash

echo ""
echo "Hi From Danny"
echo ""

mkdir -p /etc/i3

# download i3 config
until $(curl -fsSL https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/setupi3config > /etc/i3/config); do
	echo "Unable to download pi setup server"
	sleep 10
done

# download champagne
until $(curl -fsSL https://github.com/byuoitav/flight-deck/releases/download/v0.1.5/pi.tar.gz > /tmp/pi.tar.gz); do
	echo "Unable to download pi setup server"
	sleep 10
done

tar -C /tmp -xzmf /tmp/pi.tar.gz

# log to champagne.log
exec > /tmp/champagne.log 2>&1

cd /tmp && ./pi &

sleep 4
startx

# if [ ! -f "/byu/.first-boot-done" ]; then
# 	echo "First boot."
#
# 	until $(curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/first-boot.sh > /tmp/first-boot.sh); do
# 		echo "Unable to download first-boot.sh - Trying again in 5 seconds."
#         sleep 5
# 	done
# 	chmod +x /tmp/first-boot.sh
#
# 	/tmp/first-boot.sh
# else
# 	echo "Second boot."
#
# 	until $(curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/second-boot.sh > /tmp/second-boot.sh); do
# 		echo "Unable to download second-boot.sh - Trying again in 5 seconds."
#         sleep 5
# 	done
# 	chmod +x /tmp/second-boot.sh
#
# 	/tmp/second-boot.sh
# fi
