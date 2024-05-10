//go:build !minimal

package container

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type NatsContainer struct {
	Container testcontainers.Container
	URI       string
}

func StartNatsContainer(ctx context.Context) (*NatsContainer, error) {
	req := testcontainers.ContainerRequest{
		Name:         "nats-test",
		Image:        "nats:2.10.14",
		ExposedPorts: []string{"4222/tcp"},
		WaitingFor:   wait.ForExposedPort(),
		// WaitingFor:   wait.ForLog("Server is ready"),
		Cmd: []string{"-js"},
	}
	natsC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}
	ip, err := natsC.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := natsC.MappedPort(ctx, "4222")
	if err != nil {
		return nil, err
	}
	uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())
	return &NatsContainer{
		Container: natsC,
		URI:       uri,
	}, nil
}
