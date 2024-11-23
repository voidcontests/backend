#!/bin/bash

if [ "$1" == "deploy" ]; then
	if [ "$2" == "master" ]; then
		git checkout master
	fi

	echo "Deploying server from $(git rev-parse --abbrev-ref HEAD) branch"

	echo "Pulling latest image..."
	docker pull jus1d/cascade-server:latest

	echo "Starting docker compose..."
	docker compose up -d

	echo "Server running"
elif [ "$1" == "push" ]; then
    echo "Pulling latest changes..."
	git pull

	echo "Build a docker image"
	docker build -t jus1d/cascade-server:latest .

	echo "Push built docker image to docker containers registry"
	docker push jus1d/cascade-server:latest

	echo "Build docker image successfully pushed to docker containers registry"
else
	out=".bin/server"
	go build -o ${out} cmd/server/main.go

	echo "Server successfully built into \`${out}\`"
fi
