package config

func NewPostOnboardConfig(iabCredDir, sourceURL, uuid string, targets []TargetHost) Config {
	return Config{
		IAB: IAB{
			IABCredDir:      iabCredDir,
			UUID:            uuid,
			StopInstance:    false,
			HealthchecksURL: "",
		},
		Source: SourceHost{
			Name: "",
			URL:  sourceURL,
		},
		Targets:  targets,
		Projects: []Project{},
		Retention: RetentionConfig{
			Source: HostRetention{
				Default:  "",
				Projects: map[string]ProjectRetention{},
			},
			Targets: map[string]HostRetention{},
		},
	}
}
