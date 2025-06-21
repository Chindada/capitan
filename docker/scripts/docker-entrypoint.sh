#!/bin/bash
set -e

$ROOT_PATH/bin/capitan &
while ! nc -z localhost $SRV_PORT; do
    sleep 0.3
done
$ROOT_PATH/proxy/sbin/proxy

# Wait all process to exit
wait
exit
