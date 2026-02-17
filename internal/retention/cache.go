package retention

import "sync"

var scheduleCache sync.Map // map[string]Schedule

func ParseScheduleCached(s string) (Schedule, error) {
	if v, ok := scheduleCache.Load(s); ok {
		return v.(Schedule), nil
	}
	parsed, err := ParseSchedule(s)
	if err != nil {
		return Schedule{}, err
	}
	scheduleCache.Store(s, parsed)
	return parsed, nil
}
