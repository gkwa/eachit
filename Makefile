ifeq ($(OS),Windows_NT)
    SOURCES := $(shell dir /S /B *.go)
else
    SOURCES := $(shell find . -name '*.go')
endif

ifeq ($(shell uname),Darwin)
    GOOS = darwin
    GOARCH = amd64
    EXEEXT =
else ifeq ($(shell uname),Linux)
    GOOS = linux
    GOARCH = amd64
    EXEEXT =
else ifeq ($(OS),Windows_NT)
    GOOS = windows
    GOARCH = amd64
    EXEEXT = .exe
endif

APP := eachit$(EXEEXT)
VERSION := $(shell git describe --tags --dirty --always)
TARGET := ./dist/eachit_$(GOOS)_$(GOARCH)_$(VERSION)/$(APP)

$(APP): $(TARGET)
	cp $< $@

$(TARGET): $(SOURCES)
	gofumpt -w $(SOURCES)
	goreleaser build --single-target --snapshot --clean
	go vet ./...

all:
	goreleaser build --snapshot --clean

.PHONY: clean
clean:
	rm -f eachit
	rm -f $(TARGET)
	rm -rf dist
