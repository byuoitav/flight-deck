#!/usr/bin/env bash

echo "Second boot."

# wait for a deployment
until [ -f "/tmp/deployment.log" ]; do
    echo "Use the av-cli to float to $(hostname)"
    sleep 10
done

printf "\nYay! I'm floating!\n"

source /etc/environment

# Wait for Salt Env Vars to be set if not already
while [ -z $SALT_MASTER_HOST ]; do
    echo "Waiting for Salt environment variables to be set"
    sleep 5
    source /etc/environment
done

echo "master: $SALT_MASTER_HOST" > /etc/salt/minion
echo "master_finger: $SALT_MASTER_FINGER" >> /etc/salt/minion
echo "startup_states: 'highstate'" >> /etc/salt/minion

# setup minion id
# echo $SYSTEM_ID > /etc/salt/minion_id

# make changes take effect
echo "Starting salt highstate; This should take ~1 minutes"
systemctl restart salt-minion
sleep 60

# docker
echo "Waiting for deployment to finish (~3 minutes)"
sleep 180

shutdown -r

systemctl disable pi-setup.service
rm -f /byu/.first-boot-done
rm -f /byu/setup-started

echo "Setup complete! I'll never see you again."
echo "I'll reboot soon."
sleep 8

echo ""
echo "bye lol"
sleep 2
