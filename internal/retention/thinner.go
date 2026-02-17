package retention

import (
	"sort"
	"time"
)

type Entry struct {
	Name string
	Time time.Time
}

func Thin(entries []Entry, sched Schedule, now time.Time, pinned map[string]struct{}) (keep []Entry, remove []Entry) {
	if len(entries) == 0 {
		return nil, nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Time.Before(entries[j].Time)
	})

	alwaysKeep := map[string]struct{}{}
	if sched.AlwaysKeep > 0 && len(entries) > sched.AlwaysKeep {
		for _, e := range entries[len(entries)-sched.AlwaysKeep:] {
			alwaysKeep[e.Name] = struct{}{}
		}
	} else if sched.AlwaysKeep > 0 {
		for _, e := range entries {
			alwaysKeep[e.Name] = struct{}{}
		}
	}

	seen := make(map[time.Duration]map[int64]struct{}, len(sched.Rules))
	for _, r := range sched.Rules {
		if _, ok := seen[r.Period]; !ok {
			seen[r.Period] = make(map[int64]struct{})
		}
	}

	// "reverse loop", to keep newest and not oldest snapshot per bucket
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]

		if _, ok := alwaysKeep[e.Name]; ok {
			keep = append(keep, e)
			continue
		}
		if pinned != nil {
			if _, ok := pinned[e.Name]; ok {
				keep = append(keep, e)
				continue
			}
		}

		age := now.Sub(e.Time)
		keepByRule := false

		ts := e.Time.Unix()
		for _, r := range sched.Rules {
			if age < 0 {
				keepByRule = true
				break
			}
			if age > r.TTL {
				continue
			}

			periodSec := int64(r.Period / time.Second)
			if periodSec <= 0 {
				continue
			}
			bucket := ts / periodSec

			if _, ok := seen[r.Period][bucket]; !ok {
				seen[r.Period][bucket] = struct{}{}
				keepByRule = true
			}
		}

		if keepByRule {
			keep = append(keep, e)
		} else {
			remove = append(remove, e)
		}
	}

	return keep, remove
}
