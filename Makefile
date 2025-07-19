XGOARCH := amd64
XGOOS := linux
XBIN := $(XGOOS)_$(XGOARCH)/lftpq

all: lint test install

test:
	go test ./...

vet:
	go vet ./...

checkfmt:
	@sh -c "test -z $$(gofmt -l .)" || { echo "one or more files need to be formatted: try make fmt to fix this automatically"; exit 1; }

lint: checkfmt vet

install:
	go install ./...

xinstall:
	env GOOS=$(XGOOS) GOARCH=$(XGOARCH) go install ./...

publish:
ifndef DEST_PATH
	$(error DEST_PATH must be set when publishing)
endif
	rsync -a $(GOPATH)/bin/$(XBIN) $(DEST_PATH)/$(XBIN)
	@sha256sum $(GOPATH)/bin/$(XBIN)
