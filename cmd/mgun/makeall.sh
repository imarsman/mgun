#!/bin/bash

env GOOS=linux GOARCH=amd64 go build -o mgun_linux .
env GOOS=darwin GOARCH=amd64 go build -o mgun_macos .
env GOOS=windows GOARCH=amd64 go build -o mgun_windows .
