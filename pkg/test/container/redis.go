//go:build !minimal

package container

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type RedisContainer struct {
	Container testcontainers.Container
	URI       string
}

func StartRedisContainer(ctx context.Context) (*RedisContainer, error) {
	req := testcontainers.ContainerRequest{
		Name:         "redis-test",
		Image:        "redis:7.2",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForExposedPort(),
		// WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}
	ip, err := redisC.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := redisC.MappedPort(ctx, "6379")
	if err != nil {
		return nil, err
	}
	uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())
	return &RedisContainer{
		Container: redisC,
		URI:       uri,
	}, nil

}
