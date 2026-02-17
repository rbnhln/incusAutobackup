package config

type RetentionKind string

const (
	RetentionInstances RetentionKind = "instances"
	RetentionVolumes   RetentionKind = "volumes"
)

func (c Config) ResolveRetention(role, project string, kind RetentionKind, name string) string {
	var retentionPolicy string

	hr, ok := c.Retention.Hosts[role]
	if ok && hr.Default != "" {
		retentionPolicy = hr.Default
	}

	pr, ok := hr.Projects[project]
	if ok && pr.Default != "" {
		retentionPolicy = pr.Default
	}

	switch kind {
	case RetentionInstances:
		if pr.Instances.Default != "" {
			retentionPolicy = pr.Instances.Default
		}
		if pr.Instances.ByName != nil {
			if v, ok := pr.Instances.ByName[name]; ok && v != "" {
				retentionPolicy = v
			}
		}
	case RetentionVolumes:
		if pr.Volumes.Default != "" {
			retentionPolicy = pr.Volumes.Default
		}
		if pr.Volumes.ByName != nil {
			if v, ok := pr.Volumes.ByName[name]; ok && v != "" {
				retentionPolicy = v
			}
		}
	}

	return retentionPolicy
}
