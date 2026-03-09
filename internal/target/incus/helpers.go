package incus

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

// create deep copy for "security" reasons, no changes on original project
func cloneDevices(in map[string]map[string]string) map[string]map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]map[string]string, len(in))
	for devName, dev := range in {
		if dev == nil {
			out[devName] = nil
			continue
		}
		devCopy := make(map[string]string, len(dev))
		for k, v := range dev {
			devCopy[k] = v
		}
		out[devName] = devCopy
	}
	return out
}

func applyTargetPoolToRootDisk(devices map[string]map[string]string, pool string) {
	for devName, dev := range devices {
		if dev == nil {
			continue
		}
		if dev["type"] == "disk" && dev["path"] == "/" {
			dev["pool"] = pool
			devices[devName] = dev
			return
		}
	}
	if devices["root"] == nil {
		devices["root"] = map[string]string{}
	}
	devices["root"]["type"] = "disk"
	devices["root"]["path"] = "/"
	devices["root"]["pool"] = pool
}

func sanitizeDevicesForTarget(logger *slog.Logger, target incus.InstanceServer, devices map[string]map[string]string, excludeDevices []string) error {
	ex := make(map[string]struct{}, len(excludeDevices))
	for _, n := range excludeDevices {
		if n == "" {
			continue
		}
		ex[n] = struct{}{}
	}

	for devName, dev := range devices {
		if dev == nil {
			continue
		}

		// first use devices from exclude list
		_, ok := ex[devName]
		if ok {
			logger.Warn("dropping device due to excludeDevices config", "device", devName)
			delete(devices, devName)
			continue
		}

		// search for nics to be droped
		if dev["type"] == "nic" && dev["network"] != "" {
			netName := dev["network"]
			_, _, err := target.GetNetwork(netName)
			if err == nil {
				continue
			}
			if isNotFound(err) {
				logger.Warn("dropping nic device due to target network missing", "device", devName, "network", netName)
				delete(devices, devName)
				continue
			}
			return fmt.Errorf("check target network %s failed: %w", netName, err)
		}

		// search for additional volumes which are not present on the target host
		// if the pool is provided via config it should be sync before automatically
		if dev["type"] == "disk" && dev["pool"] != "" && dev["source"] != "" && dev["path"] != "/" {
			pool := dev["pool"]
			vol := dev["source"]

			_, _, err := target.GetStoragePoolVolume(pool, "custom", vol)
			if err == nil {
				continue
			}
			if isNotFound(err) {
				logger.Warn("dropping disk device due to absence on target host", "device", devName, "pool", pool, "volume", vol, "path", dev["path"])
				delete(devices, devName)
				continue
			}
			return fmt.Errorf("check target volume %s/%s failed: %w", pool, vol, err)
		}
	}
	return nil
}

func isNotFound(err error) bool {
	var stErr api.StatusError
	return errors.As(err, &stErr) && stErr.Status() == 404
}

func splitPoolVolume(subject string) (string, string, error) {
	parts := strings.SplitN(subject, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid volume subject %q, expected pool/volume", subject)
	}
	return parts[0], parts[1], nil
}

func stripAfterLastSlash(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 && i < len(s)-1 {
		return s[i+1:]
	}
	return s
}

func projectOrErr(projectName string) (string, error) {
	p := strings.TrimSpace(projectName)
	if p == "" {
		return "", fmt.Errorf("project name must not be empty")
	}
	return p, nil
}
