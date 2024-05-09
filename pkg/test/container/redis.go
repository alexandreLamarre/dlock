package container

import (
	"context"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type RedisContainer struct {
	Container testcontainers.Container
	URI       string
}

func StartRedisContainer(ctx context.Context) (*RedisContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:6.2",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
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

func TestWithRedis(t *testing.T) {
	ctx := context.Background()
	redis, err := StartRedisContainer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer redis.Container.Terminate(ctx)
}
