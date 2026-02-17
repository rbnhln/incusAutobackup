package retention

import (
	"fmt"
	"time"
)

type SnapshotOps struct {
	Kind   string
	List   func() ([]string, error)
	Delete func(name string) error
}

type PruneOptions struct {
	Now          time.Time
	DryRun       bool
	Pinned       map[string]struct{}
	Prefix       string
	ParseTS      func(name string) (time.Time, bool)
	RejectFuture bool
}

type PrunePlan struct {
	Keep        []Entry
	Remove      []Entry
	Unmanaged   []string
	Future      []Entry // Time.After(Now)
	ScheduleRaw string
}

func BuildPrunePlan(names []string, schedule string, opt PruneOptions) (PrunePlan, error) {
	if opt.ParseTS == nil {
		return PrunePlan{}, fmt.Errorf("ParseTS is required")
	}
	if opt.Now.IsZero() {
		opt.Now = time.Now()
	}

	// Leere Schedule => sichere Default-Policy: nichts löschen
	if schedule == "" {
		keep := make([]Entry, 0, len(names))
		unmanaged := make([]string, 0)
		future := make([]Entry, 0)

		for _, n := range names {
			if opt.Prefix != "" && (len(n) < len(opt.Prefix) || n[:len(opt.Prefix)] != opt.Prefix) {
				unmanaged = append(unmanaged, n)
				continue
			}
			if ts, ok := opt.ParseTS(n); ok {
				e := Entry{Name: n, Time: ts}
				if ts.After(opt.Now) {
					if opt.RejectFuture {
						return PrunePlan{}, fmt.Errorf("snapshot %q is in the future (%s > %s)", n, ts, opt.Now)
					}
					future = append(future, e)
				}
				keep = append(keep, e)
			} else {
				unmanaged = append(unmanaged, n)
			}
		}

		return PrunePlan{Keep: keep, Remove: nil, Unmanaged: unmanaged, Future: future, ScheduleRaw: schedule}, nil
	}

	// vorher: sched, err := ParseSchedule(schedule)
	sched, err := ParseScheduleCached(schedule)
	if err != nil {
		return PrunePlan{}, err
	}

	entries := make([]Entry, 0, len(names))
	unmanaged := make([]string, 0)
	future := make([]Entry, 0)

	for _, n := range names {
		if opt.Prefix != "" && (len(n) < len(opt.Prefix) || n[:len(opt.Prefix)] != opt.Prefix) {
			unmanaged = append(unmanaged, n)
			continue
		}

		ts, ok := opt.ParseTS(n)
		if !ok {
			unmanaged = append(unmanaged, n)
			continue
		}

		e := Entry{Name: n, Time: ts}
		if ts.After(opt.Now) {
			if opt.RejectFuture {
				return PrunePlan{}, fmt.Errorf("snapshot %q is in the future (%s > %s)", n, ts, opt.Now)
			}
			future = append(future, e)
		}

		entries = append(entries, e)
	}

	keep, remove := Thin(entries, sched, opt.Now, opt.Pinned)
	return PrunePlan{Keep: keep, Remove: remove, Unmanaged: unmanaged, Future: future, ScheduleRaw: schedule}, nil
}

func ExecutePrune(ops SnapshotOps, plan PrunePlan, opt PruneOptions) error {
	if opt.DryRun {
		return nil
	}
	if ops.Delete == nil {
		return fmt.Errorf("delete is required")
	}

	for _, e := range plan.Remove {
		if err := ops.Delete(e.Name); err != nil {
			return fmt.Errorf("delete %s snapshot %q failed: %w", ops.Kind, e.Name, err)
		}
	}
	return nil
}

func PruneSnapshots(ops SnapshotOps, schedule string, opt PruneOptions) (PrunePlan, error) {
	if ops.List == nil {
		return PrunePlan{}, fmt.Errorf("list is required")
	}

	names, err := ops.List()
	if err != nil {
		return PrunePlan{}, fmt.Errorf("list %s snapshots failed: %w", ops.Kind, err)
	}

	plan, err := BuildPrunePlan(names, schedule, opt)
	if err != nil {
		return PrunePlan{}, err
	}

	if err := ExecutePrune(ops, plan, opt); err != nil {
		return PrunePlan{}, err
	}

	return plan, nil
}
