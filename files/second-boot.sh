#!/bin/bash

echo "Second boot."

# get environment variables
echo "getting environment variables..."
until curl http://sandbag.byu.edu:2001/deploy/$(hostname); do
	echo "trying again..."
done

touch /tmp/got-pi-hostname
printf "\nrecieved env. variables\n"

# salt setup
until $(curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/salt-setup.sh > /tmp/salt-setup.sh); do
	echo "Trying again."
done
chmod +x /tmp/salt-setup.sh

until [ -f "/etc/salt/setup" ]; do
	/tmp/salt-setup.sh
	wait
done

#make sure the docker-service is enabled
sudo systemctl enable docker

# docker
until [ $(docker ps -q | wc -l) -gt 0 ]; do
	echo "Waiting for docker containers to download"
	sleep 10
done

sleep 30
