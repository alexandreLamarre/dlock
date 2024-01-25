.PHONY: build
.PHONY: testbin

GOCMD=go
BINARY_NAME=dlock
GO_BUILD_FLAGS=-o ./bin/$(BINARY_NAME)
GOBUILD=$(GOCMD) build $(GO_BUILD_FLAGS) ./cmd/dlock 
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
NATS_VERSION=v2.10.9
NATS_BIN=nats-server-$(NATS_VERSION)-$(GOOS)-$(GOARCH)
ETCD_VERSION=v3.5.11
ETCD_BIN=etcd-$(ETCD_VERSION)-$(GOOS)-$(GOARCH)

install:
	go install github.com/bufbuild/buf/cmd/buf@v1.29.0
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
	go install github.com/onsi/ginkgo/v2/ginkgo

build: gen
	$(GOBUILD) 
gen:
	buf generate
run: build
	./bin/$(BINARY_NAME)

testbin:
	wget https://github.com/nats-io/nats-server/releases/download/$(NATS_VERSION)/$(NATS_BIN).tar.gz
	tar -C ./testbin -zxvf $(NATS_BIN).tar.gz --strip-components=1 $(NATS_BIN)/nats-server 
	rm $(NATS_BIN).tar.gz
	wget https://github.com/etcd-io/etcd/releases/download/$(ETCD_VERSION)/$(ETCD_BIN).tar.gz
	tar -C ./testbin -zxvf $(ETCD_BIN).tar.gz --strip-components=1 $(ETCD_BIN)/etcd 
	rm $(ETCD_BIN).tar.gz

clean:
	rm -rf ./bin
	rm -rf ./testbin/*
	rm $(NATS_BIN).tar.gz || true
	rm $(ETCD_BIN).tar.gz || true