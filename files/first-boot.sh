#!/usr/bin/env bash

#############################
mkdir -p /etc/i3

# download i3 config
until $(curl -fsSL https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/setupi3config > /etc/i3/config); do
	echo "Unable to download pi setup server"
	sleep 10
done

# download champagne
until $(curl -fsSL https://github.com/byuoitav/flight-deck/releases/download/v0.1.4/pi.tar.gz > /tmp/pi.tar.gz); do
	echo "Unable to download pi setup server"
	sleep 10
done

tar -C /tmp -xzmf /tmp/pi.tar.gz

# log to champagne.log
exec > /tmp/champagne.log 2>&1

cd /tmp && ./pi &

sleep 4
startx

##############################
# ############
# # get the desired hostname
# while true; do
#     # get hostname
#     echo "Type the desired hostname of this device (E.g. ITB-1006-CP1), followed by [ENTER]:"
#     read -e desired_hostname

#     #Validate hostname
#     if [[ $desired_hostname =~ ^[A-Z,0-9]{2,}-[A-Z,0-9]+-[A-Z]+[0-9]+ ]]; then
#         break
#     else
#         echo -e "\n\nHostname invalid. Try again.\n\n"
#         continue
#     fi
# done

# # copy original dhcp file
# cp /etc/dhcpcd.conf /etc/dhcpcd.conf.other

# echo "127.0.0.1     $desired_hostname" >> /etc/hosts

# # overwrite resolv.conf
# echo "nameserver 127.0.0.1" > /etc/resolv.conf
# #############
