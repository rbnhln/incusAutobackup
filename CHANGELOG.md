# Changelog

## [1.1.0] - 2026-02-20

### Changed
- Backup run now executes in three strict phases: **Snapshot → Copy → Prune**. All snapshots (volumes and instances) are created before any copy operation starts, reducing drift between correlated resources. ([#6](https://github.com/rbnhln/incusAutobackup/issues/6))
- `SyncVolume` and `SyncInstance` have been split into separate `Snapshot*` and `Copy*` functions.
- Runner tasks split into `VolumeSnapshotTask` / `VolumeCopyTask` / `VolumePruneTask` and `InstanceSnapshotTask` / `InstanceCopyTask` / `InstancePruneTask`.
- `ExecCtx` now carries intermediate snapshot results (`VolumeSnapshots`, `InstanceSnapshots`) between phases.

### Removed
- Combined snapshot+copy flow (was the default, now replaced by phased approach).

## [1.0.0] - 2026-02-17

### Added
- Initial release of IncusAutobackup (IAB).
- Onboarding flow using Incus trust tokens (pins server cert, registers IAB client cert).
- Stateless backup run: create snapshots on source, replicate instances and custom volumes to target.
- Snapshot pruning for IAB-managed snapshots (`IAB_YYYYMMDD-HHMMSS`) based on retention policy strings.
- Retention override hierarchy: host role → project → volume/instance → byName.
- Best-effort runner with aggregated errors and `--dryRun*` flags.
- Optional Healthchecks notifications.
- Device sanitization during instance copy and optional `excludeDevices`.

### Known limitations
- Incus OS snapshot retention limitation (see README / issue #887); `--iOSfix` can force target retention to match source.
