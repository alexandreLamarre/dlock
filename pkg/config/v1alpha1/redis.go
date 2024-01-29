package v1alpha1

type RedisClientSpec struct {
	Network string `json:"network,omitempty"`
	Addr    string `json:"addr,omitempty"`
}
