package v1alpha1

type EtcdClientSpec struct {
	// List of etcd endpoints to connect to.
	Endpoints []string `json:"endpoints,omitempty" toml:"endpoints"`
	// Configuration for etcd client-cert auth.
	Certs *MTLSSpec `json:"certs,omitempty" toml:"certs"`
}

type MTLSSpec struct {
	// Path to the server CA certificate.
	ServerCA string `json:"serverCA,omitempty" toml:"serverCA"`
	// Path to the client CA certificate (not needed in all cases).
	ClientCA string `json:"clientCA,omitempty" toml:"clientCA"`
	// Path to the certificate used for client-cert auth.
	ClientCert string `json:"clientCert,omitempty" toml:"clientCert"`
	// Path to the private key used for client-cert auth.
	ClientKey string `json:"clientKey,omitempty" toml:"clientKey"`
}
