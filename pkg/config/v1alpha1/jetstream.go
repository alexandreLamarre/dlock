package v1alpha1

type JetStreamStorageSpec struct {
	Endpoint     string `json:"endpoint,omitempty"`
	NkeySeedPath string `json:"nkeySeedPath,omitempty"`
}
