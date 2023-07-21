BINPATH=./
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

all:
	GO111MODULE=on GOOS=$(PLATFORM) CGO_ENABLED=0 go build -gcflags "all=-N -l" -o $(BINPATH)$(BINNAME) nmid.go
