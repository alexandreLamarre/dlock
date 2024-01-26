package v1alpha1

type LockServerConfig struct {
	EtcdStorageSpec      *EtcdStorageSpec      `json:"etcd,omitempty"`
	JetStreamStorageSpec *JetStreamStorageSpec `json:"jetstream,omitempty"`
}
