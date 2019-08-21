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

# make changes take effect
systemctl restart salt-minion

echo "Starting salt highstate; This should take ~5 minutes"
salt-call state.highstate

# docker
echo "Waiting for deployment to finish (~3 minutes)"
sleep 180

systemctl disable pi-setup.service
rm /byu/.first-boot-done
rm /byu/setup-started

clear
echo "Setup complete! I'll never see you again."
echo "I'll reboot in 10 seconds."
sleep 8

echo ""
echo "bye lol"
sleep 2

reboot
