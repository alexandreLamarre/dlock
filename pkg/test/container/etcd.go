//go:build !minimal

package container

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type EtcdContainer struct {
	Container testcontainers.Container
	URI       string
}

func StartEtcdContainer(ctx context.Context) (*EtcdContainer, error) {
	// node := "0.0.0.0"
	clientPort := 2379
	node := "A"
	req := testcontainers.ContainerRequest{
		Name:         "etcd-test",
		Image:        "quay.io/coreos/etcd:v3.5.12",
		ExposedPorts: []string{"2379/tcp"},
		WaitingFor:   wait.ForExposedPort(),
		// WaitingFor:   wait.ForLog("ready to serve client requests"),
		Cmd: []string{
			"etcd",
			fmt.Sprintf("--name=%s", node),
			fmt.Sprintf("--advertise-client-urls=http://0.0.0.0:%d", clientPort),
			fmt.Sprintf("--listen-client-urls=http://0.0.0.0:%d", clientPort),
		},
		// Networks: []string{"host"},
	}
	etcdC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}
	ip, err := etcdC.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := etcdC.MappedPort(ctx, "2379")
	if err != nil {
		return nil, err
	}
	uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())
	return &EtcdContainer{
		Container: etcdC,
		URI:       uri,
	}, nil
}
