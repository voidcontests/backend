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
    echo -e "    ${bold}push${normal}               Build docker image and push to container registry"
    echo -e "    ${bold}run${normal} [${underl}env${normal}]          Run binary with provided environment (local - default)"
    echo -e "    ${bold}help${normal}               Print this help messages to standard output"
}

deploy() {
    if [ -n "$2" ]; then
		git checkout "$2"
	fi

	echo -e "Deploying ${bold}cascadecontests/server${normal} from ${bold}$(git rev-parse --abbrev-ref HEAD)${normal} branch"

	echo -e "Pulling latest image..."
	docker pull jus1d/cascade-server:latest

	echo -e "Starting docker compose..."
	docker compose up -d

	echo -e "Server running"
}

push_to_registry() {
    echo -e "Pulling latest changes..."
	git pull

	echo -e "Build a docker image"
	docker build -t jus1d/cascade-server:latest .

	echo -e "Push built docker image to docker containers registry"
	docker push jus1d/cascade-server:latest

	echo -e "Build docker image successfully pushed to docker containers registry"
}

run() {
    # if [ ! -e "$out" ]; then
    #     build
    # fi

    build

    if [ -n "$2" ]; then
        env="$2"
    else
        env="local"
	fi

	CONFIG_PATH="./config/${env}.yaml" ./.bin/server
}

build() {
    echo "Building..."
	go build -o ${out} cmd/server/main.go

	echo -e "Server successfully built into ${bold}\`${out}\`${normal}"
}

if [ "$1" == "deploy" ]; then
    deploy
elif [ "$1" == "push" ]; then
    push_to_registry
elif [ "$1" == "run" ]; then
    run
elif [ "$1" == "help" ]; then
    help
else
    build
fi
