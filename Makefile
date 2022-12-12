PROG=bin/badkend

SRCS=.

COMMIT_HASH=$(shell git rev-parse --short HEAD || echo "GitNotFound")

BUILD_DATE=$(shell date '+%Y-%m-%d %H:%M:%S')

CFLAGS = -ldflags "-s -w -X \"main.BuildVersion=${COMMIT_HASH}\" -X \"main.BuildDate=$(BUILD_DATE)\""

ifeq ($(OS),Windows_NT)
	PLATFORM=windows
else
	ifeq ($(shell uname),Darwin)
		PLATFORM=darwin
	else
		PLATFORM=linux
	endif
endif

all:
	if [ ! -d "./bin/" ]; then \
	mkdir bin; \
	fi
	GO111MODULE=on GOOS=$(PLATFORM) CGO_ENABLED=0 go build $(CFLAGS) -o $(PROG) $(SRCS)