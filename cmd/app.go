package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/rbnhln/incusAutobackup/internal/config"
	"github.com/rbnhln/incusAutobackup/internal/notifications"
	"github.com/rbnhln/incusAutobackup/internal/runner"
	source "github.com/rbnhln/incusAutobackup/internal/source/incus"
	"github.com/rbnhln/incusAutobackup/internal/transfer"
)

func (app *application) serve() (retErr error) {
	app.logger.Info("Application started")

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

	// Build plans for each backup step
	snapshotPlan := runner.Plan{}    //performed on source
	copyPlan := runner.Plan{}        // performed for each target
	sourcePrunePlan := runner.Plan{} // performed on host
	targetPrunePlan := runner.Plan{} // performed on each target

	for _, project := range app.config.Projects {
		for _, vol := range project.Volumes {
			snapshotPlan.Add(runner.VolumeSnapshotTask{
				ProjectName: project.Name,
				PoolName:    vol.Storage,
				VolumeName:  vol.Name,
			})

			copyPlan.Add(runner.VolumeCopyTask{
				ProjectName: project.Name,
				PoolName:    vol.Storage,
				VolumeName:  vol.Name,
				Mode:        project.Mode,
			})

			srcPol := app.config.ResolveRetention("source", project.Name, config.RetentionVolumes, vol.Name)
			tgtPol := app.config.ResolveRetention("target", project.Name, config.RetentionVolumes, vol.Name)
			if app.config.IAB.IncusOSfix {
				tgtPol = srcPol
			}

			sourcePrunePlan.Add(runner.VolumeSourcePruneTask{
				ProjectName:  project.Name,
				PoolName:     vol.Storage,
				VolumeName:   vol.Name,
				SourcePolicy: srcPol,
			})

			targetPrunePlan.Add(runner.VolumeTargetPruneTask{
				ProjectName:  project.Name,
				PoolName:     vol.Storage,
				VolumeName:   vol.Name,
				TargetPolicy: tgtPol,
			})
		}

		for _, inst := range project.Instances {
			snapshotPlan.Add(runner.InstanceSnapshotTask{
				ProjectName:  project.Name,
				InstanceName: inst.Name,
			})

			copyPlan.Add(runner.InstanceCopyTask{
				ProjectName:    project.Name,
				InstanceName:   inst.Name,
				Mode:           project.Mode,
				PoolName:       inst.Storage,
				ExcludeDevices: inst.ExcludeDevices,
			})

			srcPol := app.config.ResolveRetention("source", project.Name, config.RetentionInstances, inst.Name)
			tgtPol := app.config.ResolveRetention("target", project.Name, config.RetentionInstances, inst.Name)
			if app.config.IAB.IncusOSfix {
				tgtPol = srcPol
			}

			sourcePrunePlan.Add(runner.InstanceSourcePruneTask{
				ProjectName:  project.Name,
				InstanceName: inst.Name,
				SourcePolicy: srcPol,
			})

			targetPrunePlan.Add(runner.InstanceTargetPruneTask{
				ProjectName:  project.Name,
				InstanceName: inst.Name,
				TargetPolicy: tgtPol,
			})
		}
	}

	// get connection information
	sourceConfig, err := app.GetSourceHost()
	if err != nil {
		app.logger.Error("Source Host configuration missing", "error", err)
		return err
	}

	targets, err := app.GetTargets()
	if err != nil {
		app.logger.Error("Target Host configuration missing", "error", err)
		return err
	}

	// SOURCE CONNECTION
	app.logger.Info("Connecting to Source", "url", sourceConfig.URL)
	sourceClient, err := app.ConnectToSource(sourceConfig)
	if err != nil {
		app.logger.Error("Connection to source failed", "url", sourceConfig.URL, "error", err)
		return err
	}

	srcInfo, _, err := sourceClient.GetServer()
	if err != nil {
		app.logger.Error("Cannot fetch source server info", "error", err)
		return err
	}
	app.logger.Info("Source server connected",
		"name", srcInfo.Environment.ServerName,
		"version", srcInfo.Environment.ServerVersion)

	src := source.New(app.logger, sourceClient, source.Options{
		StopIfRunning: app.config.IAB.StopInstance,
	})

	// base exec conetxt contains information which is shared accross all targets
	baseExec := &runner.ExecCtx{
		Ctx:               ctx,
		Logger:            app.logger,
		Source:            src,
		DryRunCopy:        app.config.IAB.DryRunCopy,
		DryRunPrune:       app.config.IAB.DryRunPrune,
		VolumeSnapshots:   make(map[string]transfer.Artifact),
		InstanceSnapshots: make(map[string]transfer.Artifact),
	}

	// Step 1 create snapshot on source
	if err := snapshotPlan.Execute(baseExec); err != nil {
		return fmt.Errorf("snapshot phase failed: %w", err)
	}

	var allErrs []error

	// Step 2 copy/prune for each target
	for _, targetConfig := range targets {
		app.logger.Info("Connecting to target", "target", targetConfig.Name, "url", targetConfig.URL)

		targetClient, err := app.ConnectToTarget(targetConfig)
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("target %q connect failed: %w", targetConfig.Name, err))
			continue
		}

		tgtInfo, _, err := targetClient.GetServer()
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("target %q server-info failed: %w", targetConfig.Name, err))
			continue
		}

		app.logger.Info("Target server connected",
			"target", targetConfig.Name,
			"name", tgtInfo.Environment.ServerName,
			"version", tgtInfo.Environment.ServerVersion)

		tgt, err := app.NewTarget(targetConfig, targetClient)
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("target %q init failed: %w", targetConfig.Name, err))
			continue
		}

		targetExec := &runner.ExecCtx{
			Ctx:               ctx,
			Logger:            app.logger.With("target", targetConfig.Name),
			Source:            src,
			Target:            tgt,
			DryRunCopy:        app.config.IAB.DryRunCopy,
			DryRunPrune:       app.config.IAB.DryRunPrune,
			VolumeSnapshots:   baseExec.VolumeSnapshots,
			InstanceSnapshots: baseExec.InstanceSnapshots,
		}

		if err := copyPlan.Execute(targetExec); err != nil {
			allErrs = append(allErrs, fmt.Errorf("target %q copy phase failed: %w", targetConfig.Name, err))
		}

		if err := targetPrunePlan.Execute(targetExec); err != nil {
			allErrs = append(allErrs, fmt.Errorf("target %q prune phase failed: %w", targetConfig.Name, err))
		}
	}

	// Step 3 perform prune on source
	if err := sourcePrunePlan.Execute(baseExec); err != nil {
		allErrs = append(allErrs, fmt.Errorf("source prune phase failed: %w", err))
	}

	if len(allErrs) > 0 {
		return errors.Join(allErrs...)
	}
	return nil

}
