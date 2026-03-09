package api

import (
	incusAPI "github.com/lxc/incus/v6/shared/api"
)

// PlanType represents the type if plan being returned or requested via the API.
type PlanType string

// PlanTypeAny defines the plan type value for requesting any plan type.
const PlanTypeAny = PlanType("")

// PlanTypeContainer defines the plan type value for a container.
const PlanTypeBackup = PlanType("backup")

// PlanTypeVM defines the plan type value for a virtual-machine.
const PlanTypeRestore = PlanType("restore")

// PlanPut represents the modifiable fields of an plan.
//
// swagger:model
//
// API extension: plans.
type PlanPut struct {
	// A human-friendly name for this source
	// Example: MyPlan
	Name string `json:"name" yaml:"name"`

	// Plan configuration (see doc/plans.md)
	// Example: {"security.nesting": "true"}
	Config ConfigMap `json:"config" yaml:"config"`

	// Whether only the plans disk should be restored
	// Example: false
	DiskOnly bool `json:"disk_only,omitempty" yaml:"disk_only,omitempty"`

	// Plan description
	// Example: My test plan
	Description string `json:"description" yaml:"description"`

	// The data source (name) of plan
	// Example: container
	DataSource string `json:"data_source" yaml:"data_source"`

	// The data source (name) of plan
	// Example: container
	DataTarget string `json:"data_target" yaml:"data_target"`

	// List of profiles applied to the plan
	// Example: ["default"]
	//	Profiles []string `json:"profiles" yaml:"profiles"`

	// If set, plan will be restored to the provided snapshot name
	// Example: snap0
	Restore string `json:"restore,omitempty" yaml:"restore,omitempty"`

	// The type of plan (backup or restore)
	// Example: container
	Type string `json:"type" yaml:"type"`
}

// Plan represents an plan.
//
// swagger:model
//
// API extension: plans.
type Plan struct {
	PlanPut `yaml:",inline"`

	// Plan creation timestamp
	// Example: 2021-03-23T20:00:00-04:00
	//	CreatedAt time.Time `json:"created_at" yaml:"created_at"`

	// Expanded configuration (all profiles and local config merged)
	// Example: {"security.nesting": "true"}
	ExpandedConfig incusAPI.ConfigMap `json:"expanded_config,omitempty" yaml:"expanded_config,omitempty"`

	// Plan status (see plan_state)
	// Example: Running
	Status string `json:"status" yaml:"status"`

	// Plan status code (see plan_state)
	// Example: 101
	StatusCode incusAPI.StatusCode `json:"status_code" yaml:"status_code"`

	// Last start timestamp
	// Example: 2021-03-23T20:00:00-04:00
	//	LastUsedAt time.Time `json:"last_used_at" yaml:"last_used_at"`

}

// Writable converts a full Plan struct into a PlanPut struct (filters read-only fields).
//
// API extension: instances.
func (c *Plan) Writable() PlanPut {
	return c.PlanPut
}

// IsActive checks whether the instance state indicates the instance is active.
//
// API extension: instances.
func (c *Plan) IsActive() bool {
	switch c.StatusCode {
	case incusAPI.Stopped:
		return false
	case incusAPI.Error:
		return false
	default:
		return true
	}
}

/*
// URL returns the URL for the instance.
func (c *Plan) URL(apiVersion string, project string) *URL {
	return NewURL().Path(apiVersion, "instances", c.Name).Project(project)
}
*/
