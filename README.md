[![PkgGoDev](https://pkg.go.dev/badge/github.com/rbnhln/incusAutobackup)](https://pkg.go.dev/github.com/rbnhln/incusautobackup)
[![Go Report Card](https://goreportcard.com/badge/github.com/rbnhln/incusAutobackup)](https://goreportcard.com/report/github.com/rbnhln/incusAutobackup)

# IncusAutobackup (IAB)

Stateless backup tool for two Incus(OS) systems in a **source → target** topology.

- Creates snapshots on the source
- Replicates custom volumes and instances to the target
- Prunes only IAB-managed snapshots based on a retention policy

Inspired by: [psy0rz: zfs_autobackup](https://github.com/psy0rz/zfs_autobackup)

## Status / Disclaimer

This project is in an early stage. Expect bugs and sharp edges. Test thoroughly before using it on production systems.

Tested primarily with **Incus OS**, but it should also work with regular Incus setups. Make sure the Incus API is reachable. 

### Important note: Incus OS snapshot retention limitation

There is a current Incus OS [issue](https://github.com/lxc/incus-os/issues/887) affecting snapshot handling/retention in some setups.

Practical impact:

- Keeping a different amount of snapshots on source vs. target does not work as expected
- This repository includes a workaround flag `--iOSfix` (default: `true`) which forces the **target retention policy** to match the **source retention policy**.

## Install

### Download binary

Download a prebuilt binary from the GitHub Releases page and place it for example in your `$HOME`.

### Build from source

Requires a working Go toolchain.

Build for x86_64 linux platform:

```bash
make build
./iab --version
```

Or adjust the go compiler to fit your needs. 

## How to run (cron/systemd)

IAB is intended to be executed periodically (e.g. via cron or a systemd timer).

Example setup: run it inside an Incus container on one of the hosts and execute daily.

### systemd example (service + timer)

Adjust paths as needed:

- `ExecStart` should point to your `iab` binary
- `WorkingDirectory` must contain your `config.json`

`/etc/systemd/system/iab.service`

```ini
[Unit]
Description=IncusAutobackup (IAB)

[Service]
Type=oneshot
WorkingDirectory=/root/
ExecStart=/root/iab
```

`/etc/systemd/system/iab.timer`

```ini
[Unit]
Description=Run IAB daily

[Timer]
OnCalendar=*-*-* 03:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

Enable:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now iab.timer
sudo systemctl list-timers --all | grep iab
```

## Onboarding

Onboarding pins the server certificate and registers an IAB client certificate on each host using a trust token.

### 1. Create trust tokens (run on both hosts)

On each host generate a trust token (**restricted is optional**):

Restricted example:

```bash
incus config trust add iab --restricted --projects default
```

Unrestricted example:

```bash
incus config trust add iab
```

### 2. Run onboarding from your workstation / runner host

```bash
./bin/iab onboard \
	--sourceURL  "https://SOURCE:8443" \
	--sourceToken "TOKEN_FROM_SOURCE" \
	--targetURL  "https://TARGET:8443" \
	--targetToken "TOKEN_FROM_TARGET" 
```

Optional flags:

- `--iabCredDir <path>`: where credentials and pinned server certs are stored
	Default: `~/.config/incusAutobackup`
- `--outConfig <path>`: where the config.json file is stored. Default is the current directory

On success, onboarding writes an initial `config.json` into the current directory. 

## Usage

IAB reads `./config.json` from the current working directory.

Run:

```bash
./iab
```

Useful flags:

- `--dryRun` (implies both copy+prune dry-run)
- `--dryRunCopy` (skip snapshot + copy)
- `--dryRunPrune` (skip pruning)
- `--log-level debug|info|warn|error` (default: info)
- `--iOSfix=true|false` (see note above, default: true)
- `--version`

## Configuration

Example config: `config.json.example` (note: the filename is currently spelled `exmaple` in this repo).

### `iab`

- `iabCredDir`: credential directory created by onboarding
- `uuid`: set by onboarding
- `stopInstance`: if `true`, stop running instances before snapshot/copy and start them afterwards
- `healthchecksUrl`: optional Healthchecks ping URL (see below)

### `hosts`

Define exactly one `source` and one `target`:

```json
"hosts": [
	{ "name": "prod-source", "role": "source", "url": "https://192.168.1.100:8443" },
	{ "name": "prod-target", "role": "target", "url": "https://192.168.1.101:8443" }
]
```

### `projects`

Projects define what to replicate:

- `name`: Incus project name (e.g. `default`)
- `mode`: `push` (default) or `pull` (Incus copy mode)
- `instances`: list of instances/VMs
- `volumes`: list of custom volumes

Instance fields:

- `name`: instance name
- `storage`: target pool name for the root disk (IAB will set the root disk pool on the target)
- `excludeDevices` (optional): drop devices by device-name during copy

Example:

```json
"instances": [
	{
		"name": "vm1",
		"storage": "local",
		"excludeDevices": ["eth0", "custom-volume-1"]
	}
]
```

Device handling note:

- If a NIC device references a managed network that does not exist on the target, IAB may drop that NIC device during copy to avoid a hard failure.
- If a disk device references a custom volume missing on the target, IAB may drop that disk device during copy.
- Use `excludeDevices` to explicitly drop known-problematic devices.

## Retention policy

Retention policies apply to pruning of snapshots created by IAB.

### Snapshot naming / scope

- IAB creates snapshots with the prefix: `IAB_`
- Format: `IAB_YYYYMMDD-HHMMSS`
- Only snapshots with this prefix are managed/pruned by IAB.
- Other snapshots are ignored.

### Policy string format

Examples:

- `3`
- `6,1h2d,1d2w,1w3m`

Meaning:

- Optional first number: `alwaysKeep` (e.g. `3`)
  Always keep the newest **3** IAB snapshots, regardless of age.

- Then one or more thinning rules: `<period><TTL>` (e.g. `1h2d`)
  For snapshots in the last `TTL`, group them into `period`-sized time buckets and keep **at most one snapshot per bucket** (the newest one in that bucket).  
  Older snapshots (outside `TTL`) are not kept by this rule.

- Multiple rules are combined: a snapshot is kept if it is selected by **any** rule (or by `alwaysKeep`).

#### Example (how rules interact):

Policy: `6,1h2d,1d2w`

Assume you create snapshots very frequently (e.g. every 15 minutes):

- `alwaysKeep=6`:
  Keeps the 6 newest snapshots no matter what.

- `1h2d` (period=1 hour, TTL=2 days):
  Looking back 2 days from “now”, split time into 1-hour buckets and keep at most 1 snapshot per hour
  (the newest snapshot that falls into each hour).

- `1d2w` (period=1 day, TTL=2 weeks):
  Looking back 2 weeks from “now”, split time into 1-day buckets and keep at most 1 snapshot per day
  (the newest snapshot that falls into each day).

The final set of kept snapshots is the union of all selections above.
So in practice you end up with:
- the newest 6 snapshots,
- plus roughly "hourly" snapshots for the last 2 days,
- plus roughly "daily" snapshots for the last 2 weeks,
- and no additional snapshots older than 2 weeks (unless they are part of `alwaysKeep`).

#### Supported units:

- `s`, `min`, `h`, `d`, `w`, `m` (30 days), `y` (365 days)

If a policy is empty or omitted, IAB will not prune (keeps all IAB snapshots). Note: every month contains 30 days and every year consists of 365 days (not calendar-aware).

### Retention override rules

Retention can be defined per:

- specific instance/volume name (`byName`)
- kind (`instances` / `volumes`)
- project
- host role (`source`/`target`)

The list is in descending priotity order. 

See `config.json.example` for a full hierarchy.

## Notifications

### Healthchecks

Provide your [healthchecks](https://github.com/healthchecks/healthchecks) URL to enable start and finish notifications. 

## Licensing

This project is licensed under the MIT License. See `LICENSE`.

This software depends on other open-source libraries, including:

- Incus Go client library: `github.com/lxc/incus/v6` (Apache License 2.0)
	License text in this repo: `vendor/github.com/lxc/incus/v6/COPYING`
	Upstream: https://github.com/lxc/incus

- `github.com/google/uuid` (BSD-3-Clause)
	License text in this repo: `vendor/github.com/google/uuid/LICENSE`
	Upstream: https://github.com/google/uuid/blob/master/LICENSE

For a complete list of dependencies, see `go.mod` and `vendor/`.

