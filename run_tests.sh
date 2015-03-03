#!/usr/bin/env bash

# The typhon tests need a running rabbitmq, so a simple go test ./... 
# isn't sufficient

# TODO make sure crane is installed
# TODO make port rabbit runs on configurable

crane lift

# on our mac the rabbitmq will run on the boot2docker vm
# TODO find a cleaner solution for this. Perhaps use $DOCKER_HOST
if [[ `uname` == "Darwin" ]] ; then
  docker_ip=$(boot2docker ip)
else
  docker_ip=127.0.0.1
fi

rabbit_port=25672
export RABBIT_EXCHANGE=typhon_tests
export RABBIT_URL=${RABBIT_URL:-amqp://admin:guest@$docker_ip:$rabbit_port}

echo "RABBIT_URL=$RABBIT_URL"
echo "RABBIT_EXCHANGE=$RABBIT_EXCHANGE"

# TODO lock down dependencies somehow
go get ./...

go test ./...
