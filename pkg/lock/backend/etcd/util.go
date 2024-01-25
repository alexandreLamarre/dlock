package etcd

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/alexandreLamarre/dlock/pkg/config/v1alpha1"
	"github.com/alexandreLamarre/dlock/pkg/util"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func NewEtcdClient(ctx context.Context, conf *v1alpha1.EtcdStorageSpec) (*clientv3.Client, error) {
	var tlsConfig *tls.Config
	if conf.Certs != nil {
		var err error
		tlsConfig, err = util.LoadClientMTLSConfig(*conf.Certs)
		if err != nil {
			return nil, fmt.Errorf("failed to load client TLS config: %w", err)
		}
	}
	clientConfig := clientv3.Config{
		Endpoints: conf.Endpoints,
		TLS:       tlsConfig,
		Context:   context.WithoutCancel(ctx),
	}
	cli, err := clientv3.New(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}
	return cli, err
}
