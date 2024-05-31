Version := $(shell git describe --tags --dirty)
GitCommit := $(shell git rev-parse HEAD)
LDFLAGS := "-s -w -X main.Version=$(Version) -X main.GitCommit=$(GitCommit)"
BINPATH=./bin
BINNAME=nmid

ifeq ($(OS),Windows_NT)
	PLATFORM=windows
	BINNAME=nmid.exe
else
	ifeq ($(shell uname),Darwin)
		PLATFORM=darwin
	else
		PLATFORM=linux
	endif
endif

.PHONY: all
all: local dist

.PHONY: local
local:
	GO111MODULE=on GOOS=$(PLATFORM) CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -gcflags "all=-N -l" -o $(BINPATH)/local/$(BINNAME) nmid.go
	cp -r config $(BINPATH)/local/

.PHONY: dist
dist:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags $(LDFLAGS) -installsuffix cgo -o $(BINPATH)/nmid-amd64
	GOARM=7 GOARCH=arm CGO_ENABLED=0 GOOS=linux go build -a -ldflags $(LDFLAGS) -installsuffix cgo -o $(BINPATH)/nmid-arm
	GOARCH=arm64 CGO_ENABLED=0 GOOS=linux go build -a -ldflags $(LDFLAGS) -installsuffix cgo -o $(BINPATH)/nmid-arm64
	GOOS=windows CGO_ENABLED=0 go build -a -ldflags $(LDFLAGS) -installsuffix cgo -o $(BINPATH)/nmid.exe
	GOOS=darwin CGO_ENABLED=0 go build -a -ldflags $(LDFLAGS) -installsuffix cgo -o $(BINPATH)/nmid-darwin
	cp -r config $(BINPATH)/

.PHONY: clean
clean:
	rm -rf $(BINPATH)