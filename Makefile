# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
BINARY_NAME = metrics-gen
BINARY_UNIX = $(BINARY_NAME)_unix
OUTPUT_DIR = bin

ROOT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

all: clean build

build:
	mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME) -v

clean:
	$(GOCLEAN)
	rm -rf $(OUTPUT_DIR)

test:
	$(GOTEST) -v ./...

run:
	$(GOBUILD) -o $(BINARY_NAME) -v
	./$(BINARY_NAME)

deps:
	$(GOGET) github.com/example/dependency

# Cross compilation for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v

format:
	gofumpt -l -w . && \
	golines -m 90 -w .

gen-example: clean build
	for d in examples/*/ ; do \
		$(ROOT_DIR)/$(OUTPUT_DIR)/$(BINARY_NAME) generate -d $$d -i ; \
	done

build-example: gen-example
	for d in examples/*/ ; do \
		pushd $$d && \
		go build -o $(ROOT_DIR)/$(OUTPUT_DIR)/$$(basename $$d) && \
		popd ; \
	done
