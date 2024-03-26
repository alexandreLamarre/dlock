package v1alpha1

type LockServerConfig struct {
	EtcdClientSpec      *EtcdClientSpec      `json:"etcd,omitempty" toml:"etcd"`
	JetstreamClientSpec *JetstreamClientSpec `json:"jetstream,omitempty" toml:"jetstream"`
	RedisClientSpec     *RedisClientSpec     `json:"redis,omitempty" toml:"redis"`
}

type TracesConfig struct {
	// TODO
}

type MetricsConfig struct {
	// TODO
}
