#!/bin/bash

sleep 5
chvt 2

# try to download the setup script
until $(curl --insecure https://raw.githubusercontent.com/byuoitav/flight-deck/master/files/setup.sh > /tmp/setup.sh); do
    echo "Unable to download setup script; Trying again in 5 seconds."
    sleep 5
done

chmod +x /tmp/setup.sh
/tmp/setup.sh
