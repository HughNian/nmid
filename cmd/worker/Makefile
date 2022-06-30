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
	GO111MODULE=on GOOS=$(PLATFORM) CGO_ENABLED=0 go build -gcflags "all=-N -l" -o nmid-worker worker.go
