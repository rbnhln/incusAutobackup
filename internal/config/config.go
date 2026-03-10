package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rbnhln/incusAutobackup/internal/retention"
)

type IAB struct {
	IABCredDir      string `json:"iabCredDir"`
	UUID            string `json:"uuid"`
	StopInstance    bool   `json:"stopInstance,omitempty"`
	HealthchecksURL string `json:"healthchecksUrl,omitempty"`
	DryRunCopy      bool   `json:"-"`
	DryRunPrune     bool   `json:"-"`
	IncusOSfix      bool   `json:"-"`
}

type TargetType string

const (
	TargetTypeIncus TargetType = "incus"
)

type SourceHost struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url"`
}

type TargetHost struct {
	Name string     `json:"name"`
	Type TargetType `json:"type"`
	URL  string     `json:"url"`
}

type Instance struct {
	Name           string   `json:"name"`
	Storage        string   `json:"storage"`
	ExcludeDevices []string `json:"excludeDevices,omitempty"`
}

type Volume struct {
	Name    string `json:"name"`
	Storage string `json:"storage"`
}

type Project struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Mode        string     `json:"mode,omitempty"`
	Instances   []Instance `json:"instances,omitempty"`
	Volumes     []Volume   `json:"volumes,omitempty"`
}

type RetentionGroup struct {
	Default string            `json:"default,omitempty"`
	ByName  map[string]string `json:"byName,omitempty"`
}

type ProjectRetention struct {
	Default   string         `json:"default,omitempty"`
	Instances RetentionGroup `json:"instances,omitempty"`
	Volumes   RetentionGroup `json:"volumes,omitempty"`
}

type HostRetention struct {
	Default  string                      `json:"default,omitempty"`
	Projects map[string]ProjectRetention `json:"projects,omitempty"`
}

type RetentionConfig struct {
	Source  HostRetention            `json:"source,omitempty"`
	Targets map[string]HostRetention `json:"targets,omitempty"`
}

type Config struct {
	IAB       IAB             `json:"iab"`
	Source    SourceHost      `json:"source"`
	Targets   []TargetHost    `json:"targets"`
	Projects  []Project       `json:"projects"`
	Retention RetentionConfig `json:"retention,omitempty"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(file, cfg)
	if err != nil {
		return nil, err
	}

	for i := range cfg.Projects {
		if cfg.Projects[i].Mode == "" {
			cfg.Projects[i].Mode = "push"
		}

	}
	return cfg, nil
}

func Write(path string, cfg Config) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o600)
}

func (c *Config) Validate() error {
	var errs []error

	if strings.TrimSpace(c.IAB.IABCredDir) == "" {
		errs = append(errs, fmt.Errorf("iab.iabCredDir must not be empty"))
	}

	if strings.TrimSpace(c.Source.URL) == "" {
		errs = append(errs, fmt.Errorf("source.url must not be empty"))
	}

	if len(c.Targets) == 0 {
		errs = append(errs, fmt.Errorf("at least one target must be configured"))
	}

	targetNames := make(map[string]struct{}, len(c.Targets))
	for i, t := range c.Targets {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			errs = append(errs, fmt.Errorf("targets[%d].name must not be empty", i))
		} else {
			if _, exists := targetNames[name]; exists {
				errs = append(errs, fmt.Errorf("duplicate target name: %q", name))
			}
			targetNames[name] = struct{}{}
		}

		if strings.TrimSpace(t.URL) == "" {
			errs = append(errs, fmt.Errorf("targets[%d].url must not be empty", i))
		}

		switch t.Type {
		case TargetTypeIncus:
		default:
			errs = append(errs, fmt.Errorf("targets[%d].type %q is not supported", i, t.Type))
		}
	}

	if len(c.Projects) == 0 {
		errs = append(errs, fmt.Errorf("at least one project must be configured"))
	}

	projectNames := make(map[string]struct{}, len(c.Projects))
	for i, p := range c.Projects {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			errs = append(errs, fmt.Errorf("projects[%d].name must not be empty", i))
			continue
		}
		if _, exists := projectNames[name]; exists {
			errs = append(errs, fmt.Errorf("duplicate project name: %q", name))
			continue
		}
		projectNames[name] = struct{}{}
	}

	// Validate retention syntax for source + per-target blocks.
	validateHostRetention := func(scope string, hr HostRetention) {
		if hr.Default != "" {
			if _, err := retention.ParseSchedule(hr.Default); err != nil {
				errs = append(errs, fmt.Errorf("retention.%s.default: %w", scope, err))
			}
		}

		for projectName, pr := range hr.Projects {
			if pr.Default != "" {
				if _, err := retention.ParseSchedule(pr.Default); err != nil {
					errs = append(errs, fmt.Errorf("retention.%s.projects.%s.default: %w", scope, projectName, err))
				}
			}

			if pr.Instances.Default != "" {
				if _, err := retention.ParseSchedule(pr.Instances.Default); err != nil {
					errs = append(errs, fmt.Errorf("retention.%s.projects.%s.instances.default: %w", scope, projectName, err))
				}
			}
			for name, pol := range pr.Instances.ByName {
				if pol == "" {
					continue
				}
				if _, err := retention.ParseSchedule(pol); err != nil {
					errs = append(errs, fmt.Errorf("retention.%s.projects.%s.instances.byName.%s: %w", scope, projectName, name, err))
				}
			}

			if pr.Volumes.Default != "" {
				if _, err := retention.ParseSchedule(pr.Volumes.Default); err != nil {
					errs = append(errs, fmt.Errorf("retention.%s.projects.%s.volumes.default: %w", scope, projectName, err))
				}
			}
			for name, pol := range pr.Volumes.ByName {
				if pol == "" {
					continue
				}
				if _, err := retention.ParseSchedule(pol); err != nil {
					errs = append(errs, fmt.Errorf("retention.%s.projects.%s.volumes.byName.%s: %w", scope, projectName, name, err))
				}
			}
		}
	}

	// Source retention block
	validateHostRetention("source", c.Retention.Source)

	// Target retention blocks
	for targetName := range c.Retention.Targets {
		if _, ok := targetNames[targetName]; !ok {
			errs = append(errs, fmt.Errorf("retention.targets.%s is configured but no such target exists", targetName))
		}
	}
	for targetName, hr := range c.Retention.Targets {
		validateHostRetention(fmt.Sprintf("targets.%s", targetName), hr)
	}

	// Validate fully-resolved policies for configured resources.
	for _, p := range c.Projects {
		for _, vol := range p.Volumes {
			pol := c.ResolveSourceRetention(vol.Name, p.Name, RetentionVolumes)
			if pol != "" {
				if _, err := retention.ParseSchedule(pol); err != nil {
					errs = append(errs, fmt.Errorf("resolved retention (source/%s volume %s): %w", p.Name, vol.Name, err))
				}
			}

			for _, t := range c.Targets {
				tpol := c.ResolveTargetRetention(t.Name, vol.Name, p.Name, RetentionVolumes)
				if tpol == "" {
					continue
				}
				if _, err := retention.ParseSchedule(tpol); err != nil {
					errs = append(errs, fmt.Errorf("resolved retention (target=%s/%s volume %s): %w", t.Name, p.Name, vol.Name, err))
				}
			}
		}

		for _, inst := range p.Instances {
			pol := c.ResolveSourceRetention(inst.Name, p.Name, RetentionInstances)
			if pol != "" {
				if _, err := retention.ParseSchedule(pol); err != nil {
					errs = append(errs, fmt.Errorf("resolved retention (source/%s instance %s): %w", p.Name, inst.Name, err))
				}
			}

			for _, t := range c.Targets {
				tpol := c.ResolveTargetRetention(t.Name, inst.Name, p.Name, RetentionInstances)
				if tpol == "" {
					continue
				}
				if _, err := retention.ParseSchedule(tpol); err != nil {
					errs = append(errs, fmt.Errorf("resolved retention (target=%s/%s instance %s): %w", t.Name, p.Name, inst.Name, err))
				}
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
