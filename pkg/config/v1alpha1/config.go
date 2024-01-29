package v1alpha1

type LockServerConfig struct {
	EtcdClientSpec      *EtcdClientSpec      `json:"etcd,omitempty"`
	JetstreamClientSpec *JetstreamClientSpec `json:"jetstream,omitempty"`
	RedisClientSpec     *RedisClientSpec     `json:"redis,omitempty"`
}

type TracesConfig struct {
	// TODO
}

type MetricsConfig struct {
	// TODO
}
