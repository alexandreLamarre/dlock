FROM golang:1.21 as builder

# Set destination for COPY
WORKDIR /usr/src/app

# Set up build dependencies
RUN go install \
github.com/bufbuild/buf/cmd/buf@v1.29.0

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY . .

# Build
RUN make build

FROM ubuntu:22.04

COPY --from=builder /usr/src/app/bin/dlock /usr/local/bin/dlock
RUN chmod +x /usr/local/bin/dlock

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/engine/reference/builder/#expose
EXPOSE 5055
ENTRYPOINT ["dlock", "--addr", "tcp4://127.0.0.1:5055"]