#!/bin/bash

# colors
bold='\033[0;1m'
italic='\033[0;3m'
underl='\033[0;4m'
red='\033[0;31m'
green='\033[0;32m'
blue='\033[0;34m'
yellow='\033[0;33m'
normal='\033[0m'

out=".bin/server"

help() {
    echo -e "${underl}Usage:${normal}\n"
    echo -e "    ${bold}$0${normal} [${underl}command${normal}]\n"
    echo -e "Here is a list of available commands\n"
    echo -e "    ${bold}deploy${normal} [${underl}branch${normal}]    Run deploy script from current or provided branch"
    echo -e "    ${bold}image${normal} [push]       Build docker image and (optional) push to container registry"
    echo -e "    ${bold}run${normal} [${underl}env${normal}]          Run binary with provided environment (local - default)"
    echo -e "    ${bold}help${normal}               Print this help messages to standard output"
}

build_executable() {
    echo "Building executable..."
	go build -o ${out} cmd/server/main.go

	echo -e "Server successfully built into ${bold}\`${out}\`${normal}"
}

if [ "$1" == "image" ]; then
    GIT_COMMIT=$(git rev-parse --short HEAD)

    echo -e "Building a docker image from commit ${bold}$GIT_COMMIT${normal}"
    docker build -t jus1d/void-server:latest .

    if [ "$2" == "push" ]; then
        docker tag jus1d/void-server:latest jus1d/void-server:$GIT_COMMIT

        tags=("$GIT_COMMIT" "latest")
        for tag in "${tags[@]}"; do
            echo -e "Pushing ${bold}jus1d/void-server:$tag${normal} to hub"
            docker push jus1d/void-server:"$tag"
        done

    	echo "Built docker image was successfully pushed to dockerhub"
    fi
elif [ "$1" == "deploy" ]; then
    if [ -n "$2" ]; then
		git checkout "$2"
	fi

	echo -e "Deploying ${bold}voidcontests/server${normal} from ${bold}$(git rev-parse --abbrev-ref HEAD)${normal} branch"

	echo "Pulling latest image..."
	docker pull jus1d/void-server:latest

	echo "Stopping docker compose..."
	docker compose down

	echo "Starting docker compose..."
	docker compose up -d

	echo "Server running"
elif [ "$1" == "run" ]; then
    build_executable

    if [ -n "$2" ]; then
        env="$2"
    else
        env="local"
	fi

	if [ $env == "local" ]; then
	   echo "Start docker containers with environment"
	   docker compose -f ./docker-compose.local.yaml up -d
	fi

	CONFIG_PATH="./config/${env}.yaml" ./.bin/server

	echo "Shutting down environment containers"
	docker compose down
elif [ "$1" == "help" ]; then
    help
else
    build
fi
