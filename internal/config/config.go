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

type Host struct {
	Name string `json:"name"`
	Role string `json:"role"`
	URL  string `json:"url"`
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
	Hosts map[string]HostRetention `json:"hosts,omitempty"`
}

type Config struct {
	IAB       IAB             `json:"iab"`
	Hosts     []Host          `json:"hosts"`
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

	for role, hr := range c.Retention.Hosts {
		if hr.Default != "" {
			_, err := retention.ParseSchedule(hr.Default)
			if err != nil {
				errs = append(errs, fmt.Errorf("retention.hosts.%s.default: %w", role, err))
			}
		}
		for projectName, pr := range hr.Projects {
			if pr.Default != "" {
				_, err := retention.ParseSchedule(pr.Default)
				if err != nil {
					errs = append(errs, fmt.Errorf("retention.hosts.%s.projects.%s.default: %w", role, projectName, err))
				}
			}
			if pr.Instances.Default != "" {
				_, err := retention.ParseSchedule(pr.Instances.Default)
				if err != nil {
					errs = append(errs, fmt.Errorf("retention.hosts.%s.projects.%s.instances.default: %w", role, projectName, err))
				}
			}
			for name, pol := range pr.Instances.ByName {
				if pol == "" {
					continue
				}
				_, err := retention.ParseSchedule(pol)
				if err != nil {
					errs = append(errs, fmt.Errorf("retention.hosts.%s.projects.%s.instances.byName.%s: %w", role, projectName, name, err))
				}
			}
			if pr.Volumes.Default != "" {
				_, err := retention.ParseSchedule(pr.Volumes.Default)
				if err != nil {
					errs = append(errs, fmt.Errorf("retention.hosts.%s.projects.%s.volumes.default: %w", role, projectName, err))
				}
			}
			for name, pol := range pr.Volumes.ByName {
				if pol == "" {
					continue
				}
				_, err := retention.ParseSchedule(pol)
				if err != nil {
					errs = append(errs, fmt.Errorf("retention.hosts.%s.projects.%s.volumes.byName.%s: %w", role, projectName, name, err))
				}
			}
		}
	}

	for _, p := range c.Projects {
		for _, vol := range p.Volumes {
			for _, role := range []string{"source", "target"} {
				pol := c.ResolveRetention(role, p.Name, RetentionVolumes, vol.Name)
				if pol == "" {
					continue
				}
				_, err := retention.ParseSchedule(pol)
				if err != nil {
					errs = append(errs, fmt.Errorf("resolved retention (%s/%s volume %s): %w", role, p.Name, vol.Name, err))
				}
			}
		}
		for _, inst := range p.Instances {
			for _, role := range []string{"source", "target"} {
				pol := c.ResolveRetention(role, p.Name, RetentionInstances, inst.Name)
				if pol == "" {
					continue
				}
				_, err := retention.ParseSchedule(pol)
				if err != nil {
					errs = append(errs, fmt.Errorf("resolved retention (%s/%s instance %s): %w", role, p.Name, inst.Name, err))
				}
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
