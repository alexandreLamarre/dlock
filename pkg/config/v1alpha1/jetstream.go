package v1alpha1

type JetstreamClientSpec struct {
	Endpoint     string `json:"endpoint,omitempty"`
	NkeySeedPath string `json:"nkeySeedPath,omitempty"`
}
