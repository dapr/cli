package standalone
type mtlsConfig struct {
	Spec struct {
		MTLS struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"mtls"`
	} `yaml:"spec"`
}
