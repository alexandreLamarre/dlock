.PHONY: build
.PHONY: testbin

GOCMD=go
GO_BUILD_FLAGS=-v -tags "minimal redis"
GO_TEST_FLAGS=-race -tags redis,etcd,nats
GOBUILDSERVER=$(GOCMD) build $(GO_BUILD_FLAGS) -ldflags "-w -s" -o ./bin/dlock ./cmd/dlock 
GOBUILDCLI=$(GOCMD) build $(GO_BUILD_FLAGS) -ldflags "-w -s" -o ./bin/dlockctl ./cmd/dlockctl
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
NATS_VERSION=v2.10.9
NATS_BIN=nats-server-$(NATS_VERSION)-$(GOOS)-$(GOARCH)
ETCD_VERSION=v3.5.11
ETCD_BIN=etcd-$(ETCD_VERSION)-$(GOOS)-$(GOARCH)
REDIS_VERSION=7.2.0
REDIS_BIN=redis-stack-server-$(REDIS_VERSION)-v8-x86_64.AppImage

install:
	go install github.com/bufbuild/buf/cmd/buf@v1.29.0
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
	go install github.com/onsi/ginkgo/v2/ginkgo

build: gen
	$(GOBUILDSERVER) 
	$(GOBUILDCLI) 

gen:
	buf generate

run: build
	./bin/$(BINARY_NAME)

test:
	ginkgo $(GO_TEST_FLAGS) ./...