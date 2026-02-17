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
