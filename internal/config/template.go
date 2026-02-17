package config

func NewPostOnboardConfig(iabCredDir, sourceURL, targetURL, uuid string) Config {
	return Config{
		IAB: IAB{
			IABCredDir:      iabCredDir,
			UUID:            uuid,
			StopInstance:    false,
			HealthchecksURL: "",
		},
		Hosts: []Host{
			{Role: "source", URL: sourceURL, Name: ""},
			{Role: "target", URL: targetURL, Name: ""},
		},
		Projects: []Project{},
		Retention: RetentionConfig{
			Hosts: map[string]HostRetention{
				"source": {
					Default:  "", // z.B. "6,1h2d,1d2w,1w3m"
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
