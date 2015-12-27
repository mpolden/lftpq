all: deps test vet install

fmt:
	go fmt ./...

test:
	go test ./...

vet:
	go vet 2> /dev/null; if [ $$? -eq 3 ]; then \
		go get -v golang.org/x/tools/cmd/vet; \
	fi
	go vet ./...

deps:
	go get -d -v ./...

install:
	go install
