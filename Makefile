NAME=lftpq

all: deps test install

fmt:
	go fmt ./...

test:
	go test ./...

deps:
	go get -d -v ./...

install:
	go install
