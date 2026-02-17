package backup

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

func SyncInstance(logger *slog.Logger, source, target incus.InstanceServer, instanceName, projectMode, targetPool string, stopIfRunning bool, excludeDevices []string) error {
	logger.With("instance", instanceName)

	// 1 Check if instance exists
	inst, _, err := source.GetInstance(instanceName)
	if err != nil {
		return fmt.Errorf("get source instance %s failed: %w", instanceName, err)
	}

	// 2 Optional: Stop if running
	wasRunning := false
	if stopIfRunning {
		state, _, err := source.GetInstanceState(instanceName)
		if err != nil {
			return fmt.Errorf("get instance state %s failed: %w", instanceName, err)
		}

		if state != nil && state.Status == "Running" {
			wasRunning = true
			logger.Info("stopping instance for snapshot/copy")

			op, err := source.UpdateInstanceState(instanceName, api.InstanceStatePut{
				Action:  "stop",
				Timeout: 300,
				Force:   false,
			}, "")
			if err != nil {
				return fmt.Errorf("stop instance %s failed: %w", instanceName, err)
			}
			err = op.Wait()
			if err != nil {
				return fmt.Errorf("stop instance %s operation failed: %w", instanceName, err)
			}
		}
	}
	if stopIfRunning && wasRunning {
		defer func() {
			logger.Info("starting instance after snapshot/copy")
			op, err := source.UpdateInstanceState(instanceName, api.InstanceStatePut{
				Action:  "start",
				Timeout: 300,
			}, "")
			if err != nil {
				logger.Error("failed to start instance after snapshot/copy", "error", err)
				return
			}
			err = op.Wait()
			if err != nil {
				logger.Error("start instance operation failed after snapshot/copy", "error", err)
			}
		}()
	}

	// 3 Create Snapshot
	snapshotName := fmt.Sprintf("IAB_%s", time.Now().Format("20060102-150405"))
	logger.Info("creating instance snapshot", "snapshot", snapshotName)

	opSnap, err := source.CreateInstanceSnapshot(instanceName, api.InstanceSnapshotsPost{
		Name:     snapshotName,
		Stateful: false,
	})
	if err != nil {
		return fmt.Errorf("create snapshot for instance %s failed: %w", instanceName, err)
	}
	err = opSnap.Wait()
	if err != nil {
		return fmt.Errorf("create snapshot for instance %s operation failed: %w", instanceName, err)
	}

	// 4 Copy to target
	logger.Info("copying instance to target")

	copyArgs := incus.InstanceCopyArgs{
		Name:                instanceName,
		Mode:                projectMode,
		InstanceOnly:        false,
		Refresh:             true,
		Live:                false,
		RefreshExcludeOlder: false,
		AllowInconsistent:   false,
	}

	// 4.1 Need to change the storage pool, did not find Pool flag in copy args
	// 4.2 filter devices which are not present on the target host
	instCopy := *inst
	instCopy.Devices = cloneDevices(inst.Devices)

	// root disk change
	if targetPool != "" {
		applyTargetPoolToRootDisk(instCopy.Devices, targetPool)
	}

	// sanitize devices for target host, drop with warn if not present
	err = sanitizeDevicesForTarget(logger, target, instCopy.Devices, excludeDevices)
	if err != nil {
		return fmt.Errorf("sanitize devices failed: %w", err)
	}

	//4.2 Perform Copy

	opCopy, err := target.CopyInstance(source, instCopy, &copyArgs)
	if err != nil {
		return fmt.Errorf("copy instance %s to target failed: %w", instanceName, err)
	}
	err = opCopy.Wait()
	if err != nil {
		return fmt.Errorf("copy instance %s operation failed: %w", instanceName, err)
	}

	logger.Info("instance sync successful")
	return nil
}

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
