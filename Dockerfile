FROM golang:1.26.0-alpine3.23 as builder
ARG tags

# Set destination for COPY
WORKDIR /usr/src/app

RUN apk add --no-cache make
# Set up build dependencies
RUN go install \
github.com/bufbuild/buf/cmd/buf@latest

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GO_BUILD_TAGS=${tags} make build

FROM alpine:3.23
LABEL org.opencontainers.image.source="https://github.com/alexandreLamarre/dlock"
LABEL org.opencontainers.image.description="Reliable & scalable distributed locking, scheduling and synchronization"
LABEL org.opencontainers.image.licenses="Apache-2.0"

COPY --from=builder /usr/src/app/bin/dlock /usr/local/bin/dlock
COPY --from=builder /usr/src/app/bin/dlockctl /usr/local/bin/dlockctl
RUN export PATH=$PATH:/usr/local/bin/dlockctl
RUN chmod +x /usr/local/bin/dlock
RUN chmod +x /usr/local/bin/dlockctl

EXPOSE 5055
ENTRYPOINT ["dlock", "--addr", "tcp4://127.0.0.1:5055"]