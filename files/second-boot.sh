#!/usr/bin/env bash

echo "Second boot."

# Wait for Salt Env Vars to be set if not already
while [ -z $SALT_MASTER_HOST ]; do
    echo "Waiting for Salt environment variables to be set"
    sleep 5
    source /etc/environment
done
