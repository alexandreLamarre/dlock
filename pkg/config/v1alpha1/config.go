package v1alpha1

type LockServerConfig struct {
	EtcdStorageSpec      *EtcdStorageSpec      `json:"etcdStorageSpec,omitempty"`
	JetStreamStorageSpec *JetStreamStorageSpec `json:"jetStreamStorageSpec,omitempty"`
}
