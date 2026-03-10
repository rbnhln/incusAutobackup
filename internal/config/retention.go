package config

type RetentionKind string

const (
	RetentionInstances RetentionKind = "instances"
	RetentionVolumes   RetentionKind = "volumes"
)

func (c Config) resolveFromHostRetention(hr HostRetention, project string, kind RetentionKind, name string) string {
	var retentionPolicy string

	if hr.Default != "" {
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
		v, ok := pr.Instances.ByName[name]
		if ok && v != "" {
			retentionPolicy = v
		}
	case RetentionVolumes:
		if pr.Volumes.Default != "" {
			retentionPolicy = pr.Volumes.Default
		}
		v, ok := pr.Volumes.ByName[name]
		if ok && v != "" {
			retentionPolicy = v
		}
	}

	return retentionPolicy
}

func (c Config) ResolveSourceRetention(project, name string, kind RetentionKind) string {
	return c.resolveFromHostRetention(c.Retention.Source, project, kind, name)
}

func (c Config) ResolveTargetRetention(targetName, project, name string, kind RetentionKind) string {
	hr, ok := c.Retention.Targets[targetName]
	if !ok {
		return ""
	}
	return c.resolveFromHostRetention(hr, project, kind, name)
}
