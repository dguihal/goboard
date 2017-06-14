#!/bin/bash
#set -e
set -x
cd /go/bin/
# if $proxy_domain is not set, then default to $HOSTNAME
# export MaxHistorySize=${MaxHistorySize:-50}

# ensure the following environment variables are set. exit script and container if not set.
test $MaxHistorySize
echo $MaxHistorySize

echo "Generating conf"
/usr/local/bin/confd -onetime -backend env

echo "Starting Goboard"
./app

