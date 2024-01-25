.PHONY: build

GOCMD=go
BINARY_NAME=dlock
GO_BUILD_FLAGS=-v -o ./bin/$(BINARY_NAME)
GOBUILD=$(GOCMD) build $(GO_BUILD_FLAGS) ./cmd/dlock 


build: gen
	$(GOBUILD) 
gen:
	buf generate
run: build
	./bin/$(BINARY_NAME)