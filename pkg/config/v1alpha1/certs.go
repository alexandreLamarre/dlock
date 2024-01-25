package v1alpha1

type CertsSpec struct {
	// Path to a PEM encoded CA certificate file. Mutually exclusive with CACertData
	CACert *string `json:"caCert,omitempty"`
	// String containing PEM encoded CA certificate data. Mutually exclusive with CACert
	CACertData []byte `json:"caCertData,omitempty"`
	// Path to a PEM encoded server certificate file. Mutually exclusive with ServingCertData
	ServingCert *string `json:"servingCert,omitempty"`
	// String containing PEM encoded server certificate data. Mutually exclusive with ServingCert
	ServingCertData []byte `json:"servingCertData,omitempty"`
	// Path to a PEM encoded server key file. Mutually exclusive with ServingKeyData
	ServingKey *string `json:"servingKey,omitempty"`
	// String containing PEM encoded server key data. Mutually exclusive with ServingKey
	ServingKeyData []byte `json:"servingKeyData,omitempty"`
}
