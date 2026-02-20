package main

import (
	"context"
	"fmt"

	"github.com/lxc/incus/v6/shared/api"
	"github.com/rbnhln/incusAutobackup/internal/config"
	"github.com/rbnhln/incusAutobackup/internal/notifications"
	"github.com/rbnhln/incusAutobackup/internal/runner"
)

func (app *application) serve() (retErr error) {
	app.logger.Info("Application started")

	// if configTypeBytes, err := json.MarshalIndent(app.config, "", "  "); err == nil {
	// 	fmt.Printf("\n--- LOADED CONFIG ---\n%s\n---------------------\n\n", string(configTypeBytes))
	// }

	err := app.config.Validate()
	if err != nil {
		return err
	}

	fmt.Printf("Config validated!\n")

	// for notifications
	ctx := context.Background()
	notif := notifications.NewManagerFromConfig(app.logger, app.config)
	notif.Start(ctx)
	defer func() {
		ok := retErr == nil
		_ = notif.Finish(ctx, ok)
	}()

	plan := runner.Plan{}

	// Phase 1: All Snapshots
	for _, project := range app.config.Projects {
		for _, vol := range project.Volumes {
			plan.Add(runner.VolumeSnapshotTask{
				ProjectName: project.Name,
				PoolName:    vol.Storage,
				VolumeName:  vol.Name,
			})
		}
		for _, inst := range project.Instances {
			plan.Add(runner.InstanceSnapshotTask{
				ProjectName:  project.Name,
				InstanceName: inst.Name,
			})
		}
	}

	// Phase 2: All Copies
	for _, project := range app.config.Projects {
		for _, vol := range project.Volumes {
			plan.Add(runner.VolumeCopyTask{
				ProjectName: project.Name,
				PoolName:    vol.Storage,
				VolumeName:  vol.Name,
				Mode:        project.Mode,
			})
		}
		for _, inst := range project.Instances {
			plan.Add(runner.InstanceCopyTask{
				ProjectName:    project.Name,
				InstanceName:   inst.Name,
				Mode:           project.Mode,
				PoolName:       inst.Storage,
				ExcludeDevices: inst.ExcludeDevices,
			})
		}
	}

	// Phase 3: All Prunes
	for _, project := range app.config.Projects {
		for _, vol := range project.Volumes {
			srcPol := app.config.ResolveRetention("source", project.Name, config.RetentionVolumes, vol.Name)
			tgtPol := app.config.ResolveRetention("target", project.Name, config.RetentionVolumes, vol.Name)
			if app.config.IAB.IncusOSfix {
				tgtPol = srcPol
			}
			plan.Add(runner.VolumePruneTask{
				ProjectName:  project.Name,
				PoolName:     vol.Storage,
				VolumeName:   vol.Name,
				SourcePolicy: srcPol,
				TargetPolicy: tgtPol,
			})
		}
		for _, inst := range project.Instances {
			srcPol := app.config.ResolveRetention("source", project.Name, config.RetentionInstances, inst.Name)
			tgtPol := app.config.ResolveRetention("target", project.Name, config.RetentionInstances, inst.Name)
			if app.config.IAB.IncusOSfix {
				tgtPol = srcPol
			}
			plan.Add(runner.InstancePruneTask{
				ProjectName:  project.Name,
				InstanceName: inst.Name,
				SourcePolicy: srcPol,
				TargetPolicy: tgtPol,
			})
		}
	}

	// get connection information
	sourceConfig, err := app.GetHostByRole("source")
	if err != nil {
		app.logger.Error("Source Host configuration missing", "error", err)
		return err
	}

	targetConfig, err := app.GetHostByRole("target")
	if err != nil {
		app.logger.Error("Target Host configuration missing", "error", err)
		return err
	}

	// establish connection to source and target
	app.logger.Info("Establishing connection to IncusOS-Server")

	// SOURCE CONNECTION
	app.logger.Info("Connecting to Source", "url", sourceConfig.URL)
	sourceClient, err := app.ConnectToHost(sourceConfig)
	if err != nil {
		app.logger.Error("Connection to Source failed", "url", sourceConfig.URL, "error", err)
		return err
	}

	// TARGET CONNECTION
	app.logger.Info("Connecting to target", "url", targetConfig.URL)
	targetClient, err := app.ConnectToHost(targetConfig)
	if err != nil {
		app.logger.Error("Connection to Target failed", "url", targetConfig.URL, "error", err)
		return err
	}

	srcInfo, _, err := sourceClient.GetServer()
	if err != nil {
		app.logger.Error("Cannot fetch source server info", "error", err)
		return err
	}
	app.logger.Info("Source Server Connected",
		"name", srcInfo.Environment.ServerName,
		"version", srcInfo.Environment.ServerVersion)

	tgtInfo, _, err := targetClient.GetServer()
	if err != nil {
		app.logger.Error("Cannot fetch target server info", "error", err)
		return err
	}
	app.logger.Info("Target Server Connected",
		"name", tgtInfo.Environment.ServerName,
		"version", tgtInfo.Environment.ServerVersion)

	exec := &runner.ExecCtx{
		Ctx:               context.Background(),
		Logger:            app.logger,
		Source:            sourceClient,
		Target:            targetClient,
		DryRunCopy:        app.config.IAB.DryRunCopy,
		DryRunPrune:       app.config.IAB.DryRunPrune,
		StopInstances:     app.config.IAB.StopInstance,
		VolumeSnapshots:   make(map[string]*api.StorageVolume),
		InstanceSnapshots: make(map[string]*api.Instance),
	}
	return plan.Execute(exec)
}
