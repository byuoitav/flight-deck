#!/usr/bin/env bash

# This script is used to install and set up dependencies on a newly wiped/installed Raspberry Pi
bootfile="/byu/firstboot"
started="/byu/setup-started"

# check if script has already been started
if [ -f "$started" ]; then
	echo "setup has been started remotely"
	echo "please wait for me to reboot (~20 minutes)"
	sleep infinity
	exit
fi

# start
touch $started

ip=`ip addr | grep 'state UP' -A2 | tail -n1 | awk '{print $2}' | cut -f1  -d'/'`
echo "\n\nmy ip address: $ip\n\n"

# Update the time (from google, to ensure https works)
date -s "$(wget -qSO- --max-redirect=0 google.com 2>&1 | grep Date: | cut -d' ' -f5-8)Z"

# get the desired hostname
while  true ; do
    # get hostname
    echo "Type the desired hostname of this device (E.g. ITB-1006-CP1), followed by [ENTER]:"
    read -e desired_hostname

    #Validate hostname
    if [[ $desired_hostname =~ ^[A-Z,0-9]{2,}-[A-Z,0-9]+-[A-Z]+[0-9]+ ]]; then
        break
    else
        echo -e "\n\nHostname invalid. Try again.\n\n"
        continue
    fi
done

# get the desired
while true; do
    # get static ip
    echo "Type the desired static ip-address of this device (e.g. $ip), followed by [ENTER]:"
    read -e desired_ip

    #Validate IP
    if [[ $desired_ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        break
    else
        echo -e "\n\nInvalid IP addrses. Try again.\n\n"
        continue
    fi
done

# copy original dhcp file
cp /etc/dhcpcd.conf /etc/dhcpcd.conf.other

# setup hostname
echo $desired_hostname > /etc/hostname
echo "127.0.1.1    $desired_hostname" >> /etc/hosts

# setup static ip
echo "interface eth0" >> /etc/dhcpcd.conf
echo "static ip_address=$desired_ip/24" >> /etc/dhcpcd.conf
routers=$(echo "static routers=$desired_ip" | cut -d "." -f -3)
echo "$routers.1" >> /etc/dhcpcd.conf
echo "static domain_name_servers=127.0.0.1 10.8.0.19 10.8.0.26" >> /etc/dhcpcd.conf

# overwrite resolv.conf
echo "nameserver 127.0.0.1" > /etc/resolv.conf

rm $bootfile
reboot
