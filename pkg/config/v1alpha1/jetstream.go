package v1alpha1

type JetstreamClientSpec struct {
	Endpoint     string `json:"endpoint,omitempty" toml:"endpoint"`
	NkeySeedPath string `json:"nkeySeedPath,omitempty" toml:"nkeySeedPath"`
}
