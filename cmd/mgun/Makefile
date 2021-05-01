# Makefile to build and package application in standardized layout

appdirname := $(shell basename $(dir $(abspath $(dir $$PWD/..))))
appname := $(appdirname)

.PHONY: help
.DEFAULT_GOAL := help

# Colour codes for output
NO_C := \x1b[0m
GREEN := \x1b[32;01m
BLUE := \x1b[34;01m
TEAL := \033[36m

# Figure out dir for the makefile
MAKEFILE_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

# Figure out the rood dir for the project
PROJECT_ROOT_DIR:=$(shell cd ${MAKEFILE_DIR}; cd ./../..; pwd)


# Format 2020:10:26T03:14:23Z
build_ts=`date -u '+%Y:%m:%dT%H:%M:%SZ'`
build_date=`date +%F`

# Get revision from git
version=`git rev-list --count HEAD`

# Base directory for the application code
build_base        := ${PROJECT_ROOT_DIR}/cmd/${appname}

# The dir where builds go
build_dir         := ${PROJECT_ROOT_DIR}/build/$(appname)/

# Files used for the build
basefiles_dir     := ${PROJECT_ROOT_DIR}/bin/basefiles/${appname}

# The path to the base of the build package dir for adding flags
build_namespace := gitlab.xml.team/imarsman/mgun/cmd/mgun/internal

# Build flags built into the executable. Used in application to print out build info.
# http://stackoverflow.com/questions/11354518/golang-application-auto-build-versioning
LDFLAGS=-ldflags="\
	-X ${build_namespace}/build.Version=${version} \
	-X ${build_namespace}/build.Build=${build_ts} \
	-X ${build_namespace}/build.Platform=$(1) \
	-X ${build_namespace}/build.Architecture=$(4)"

# The Golang call to do the actual build
build_app = GOOS=$(1) GOARCH=$(2) go build \
	$(LDFLAGS) \
	-o $(build_dir)/$(appname)_$(3)_$(4)$(5) &&\
	chmod 755 $(build_dir)/$(appname)_$(3)_$(4)$(5)

# Make base directories
make_base_dirs:
	mkdir -p $(build_dir)/../builds

# Copy files to be included in the build from the basefiles directory for the project.
copy_basefiles:
	find $(basefiles_dir)/ -type f -exec cp {} $(build_dir) \;

# Print out help information. Requires a double hash commentt beside the task
# name in order for it to show up. Omitting this is a good way to not show tasks
# that are not really top-level.
help: ## Print this message
	@echo "$(GREEN)Usage:$(NO_C)"
	@echo "$(TEAL)make [target]$(NO_C)"
	@echo ""
	@echo "$(GREEN)List of targets:$(NO_C)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(TEAL)- %-15s$(NO_C) %s\n", $$1, $$2}'
	@echo
	@echo "$(GREEN)Examples$(NO_C)"
	@echo "$(TEAL)make all$(NO_C)"
	@echo "$(BLUE)Make MacOs and Linux builds:$(NO_C)"
	@echo "$(TEAL)- Builds placed in$(NO_C) ./build/builds/[architecture]/[name]"

# Get all dependencies to vendor dir prior to build.
dependencies: ## Update go.mod dependencies in vendor dir
	go mod vendor

# tar and zip up the build and move to builds directory.
package:
	cd $(build_dir)/.. && \
	tar -cvzf $(appname)-build-${build_date}.tar.gz ${appname} && \
	mv $(appname)-build-${build_date}.tar.gz builds/ &&\
	zip $(appname)-build-${build_date}.zip -r ${appname} &&\
	mv $(appname)-build-${build_date}.zip builds/	

# Clean things up
clean:
	cd $(build_dir)/.. && \
	rm -rf ${appname} && \
	mkdir ${appname}

# Build for all supported architectures
# all: make_base_dirs get-dependencies clean linux windows darwin package
all: dependencies make_base_dirs copy_basefiles linux darwin darwinsilicon awsgraviton windows package clean ## Build for all supported architectures and package as tar and zip in builds dir

##### LINUX INTEL BUILD #####
linux: ## Build linux
	$(call build_app,linux,amd64,lin,x64,)
    
##### WINDOWS INTEL BUILD #####
windows: ## Build Windows
	$(call build_app,windows,amd64,win,x64,.exe)
	
##### DARWIN (MAC) INTEL BUILD #####
darwin: ## Build MacOs Intel (Darwin)
	$(call build_app,darwin,amd64,mac,x64,)

##### DARWIN (MAC) ARM (SILICON) BUILD #####
darwinsilicon: ## Build MacOs ARM (Darwin)
	$(call build_app,darwin,arm64,mac,arm64,)

##### LINUX AWS Graviton ARM BUILD #####
awsgraviton: ## Build Linux AWS Graviton ARM
	$(call build_app,linux,arm64,lin,arm64,)

info: ## Debugging information
	@echo "appname $(appname)"
	@echo "build_dir $(build_dir)"
	@echo "basefiles_dir $(basefiles_dir)"