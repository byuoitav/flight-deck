#!/bin/bash

echo ""
echo "Hi From Danny"
echo ""

mkdir -p /etc/i3

# download i3 config
until $(curl -fksSL https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/setupi3config > /etc/i3/config); do
	echo "Unable to download i3 config file"
	sleep 10
done

# download champagne
until $(curl -fksSL https://github.com/byuoitav/flight-deck/releases/download/v0.2.4/pi.tar.gz > /tmp/pi.tar.gz); do
	echo "Unable to download champagne"
	sleep 10
done

tar -C /tmp -xzmf /tmp/pi.tar.gz

# log to champagne.log
exec > /tmp/champagne.log 2>&1

cd /tmp && ./pi &

sleep 4
startx
