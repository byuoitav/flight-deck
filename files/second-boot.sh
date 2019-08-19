#!/bin/bash

echo "Second boot."

# wait for a deployment
until [ -f "/tmp/deployment.log" ]; do
    echo "Use the av-cli to float to $(hostname)"
    sleep 10
done

printf "\nYay! I'm floating!\n"

# download generic salt-minion config file
until $(curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/minion > /etc/salt/minion); do
    echo "Unable to download salt minion file - Trying again in 5 seconds"
    sleep 5
done

# replace host/finger, setup minion id
sed -i 's/\$SALT_MASTER_HOST/'$SALT_MASTER_HOST'/' /etc/salt/minion
sed -i 's/\$SALT_MASTER_FINGER/'$SALT_MASTER_FINGER'/' /etc/salt/minion
echo $SYSTEM_ID > /etc/salt/minion_id

sudo setfacl -m u:pi:rwx /etc/salt/pki/minion/
sudo setfacl -m u:pi:rwx /etc/salt/pki/minion/*

# make changes take effect
sudo systemctl restart salt-minion

# docker
until [ $(docker ps -q | wc -l) -gt 0 ]; do
	echo "Waiting for docker containers to download"
	sleep 10
done

sleep 30

systemctl disable pi-setup.service
