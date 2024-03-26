package v1alpha1

type RedisClientSpec struct {
	Network string `json:"network,omitempty" toml:"network"`
	Addr    string `json:"addr,omitempty" toml:"addr"`
}
