#!/usr/bin/env bash

# This script is called automatically by `pi-setup.sh` to run a batch of Pi setup commands that require sudo permissions
started="/usr/local/games/setup-started"

# set up screenshutoff
curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/screenshutoff-setup.sh > /tmp/sss-setup.sh
chmod +x /tmp/sss-setup.sh
sh -c "bash /tmp/sss-setup.sh"

# Perform general updating
apt update
apt -y upgrade
apt -y dist-upgrade
apt -y autoremove
apt -y autoclean

# Install an ARM build of docker-compose
easy_install --upgrade pip
pip install docker-compose

# Configure automatic login for the `pi` user
mkdir -pv /etc/systemd/system/getty@tty1.service.d/
curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/autologin.conf > /etc/systemd/system/getty@tty1.service.d/autologin.conf
systemctl enable getty@tty1.service

# Enable SSH connections
touch /boot/ssh

# Add the `pi` user to the sudoers group
usermod -aG sudo pi

# set to update from byu servers
curl https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/ntp.conf > /etc/ntp.conf
apt -y install ntpdate
systemctl stop ntp
ntpdate-debian
systemctl start ntp
ntpq -p
