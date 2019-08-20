#!/bin/bash

echo "Second boot."

# wait for a deployment
until [ -f "/tmp/deployment.log" ]; do
    echo "Use the av-cli to float to $(hostname)"
    sleep 10
done

printf "\nYay! I'm floating!\n"

source /etc/environment

echo "master: $SALT_MASTER_HOST" > /etc/salt/minion
echo "master_finger: $SALT_MASTER_FINGER" >> /etc/salt/minion
echo "startup_states: 'highstate'" >> /etc/salt/minion

# setup minion id
echo $SYSTEM_ID > /etc/salt/minion_id

setfacl -m u:pi:rwx /etc/salt/pki/minion/
setfacl -m u:pi:rwx /etc/salt/pki/minion/*

# make changes take effect
systemctl restart salt-minion

salt-call state.highstate
until [ -f "/home/pi/.ssh/authorized_keys" ]; do
    echo "Waiting for salt high state to complete (no ..ssh/authorized_keys file found yet)"
    sleep 10
done

# docker
until [ $(docker ps -q | wc -l) -gt 0 ]; do
	echo "Waiting for docker containers to download"
	sleep 10
done

sleep 30

systemctl disable pi-setup.service
