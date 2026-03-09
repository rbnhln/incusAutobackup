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
			Hosts: map[string]HostRetention{
				"source": {
					Default:  "",
					Projects: map[string]ProjectRetention{},
				},
				"target": {
					Default:  "",
					Projects: map[string]ProjectRetention{},
				},
			},
		},
	}
}
