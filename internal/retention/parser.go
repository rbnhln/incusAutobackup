package retention

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Rule struct {
	Period time.Duration
	TTL    time.Duration
	Raw    string
}

type Schedule struct {
	AlwaysKeep int
	Rules      []Rule
	Raw        string
}

var (
	ruleRegEx = regexp.MustCompile(`^(\d+)([a-z]+)(\d+)([a-z]+)$`)
)

const IABSnapshotPrefix = "IAB_"

func ParseIABSnapshotTime(name string) (time.Time, bool) {
	if !strings.HasPrefix(name, IABSnapshotPrefix) {
		return time.Time{}, false
	}
	ts := strings.TrimPrefix(name, IABSnapshotPrefix)

	// erwartetes Format: 20060102-150405
	t, err := time.ParseInLocation("20060102-150405", ts, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func ParseSchedule(s string) (Schedule, error) {
	schedule := Schedule{Raw: strings.TrimSpace(s)}
	if schedule.Raw == "" {
		return schedule, nil
	}

	parts := strings.Split(schedule.Raw, ",")

	seenAlwaysKeep := false
	seenPeriods := map[time.Duration]string{} // period -> rawRule (für bessere Fehlermeldungen)

	for idx, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return Schedule{}, fmt.Errorf("empty element found at position %d (double comma?)", idx+1)
		}

		if isDigits(p) {
			if idx != 0 {
				return Schedule{}, fmt.Errorf("always-keep number must be the first element (got %q at position %d)", p, idx+1)
			}
			if seenAlwaysKeep {
				return Schedule{}, fmt.Errorf("always-keep specified more than once (got %q)", p)
			}

			n, err := strconv.Atoi(p)
			if err != nil {
				return Schedule{}, fmt.Errorf("invalid always-keep %q: %w", p, err)
			}
			if n < 0 {
				return Schedule{}, fmt.Errorf("always-keep must be non-negative: %d", n)
			}

			schedule.AlwaysKeep = n
			seenAlwaysKeep = true
			continue
		}

		r, err := parseRule(p)
		if err != nil {
			return Schedule{}, err
		}

		if prev, ok := seenPeriods[r.Period]; ok {
			return Schedule{}, fmt.Errorf("duplicate period %s in schedule (%q and %q)", r.Period, prev, r.Raw)
		}
		seenPeriods[r.Period] = r.Raw

		schedule.Rules = append(schedule.Rules, r)
	}

	return schedule, nil
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

func parseRule(s string) (Rule, error) {
	match := ruleRegEx.FindStringSubmatch(strings.ToLower(strings.TrimSpace(s)))
	if len(match) != 5 {
		return Rule{}, fmt.Errorf("invalid rule %q (expected e.g. 1d1w)", s)
	}

	periodAmount, _ := strconv.Atoi(match[1])
	periodUnit := match[2]
	ttlAmount, _ := strconv.Atoi(match[3])
	ttlUnit := match[4]

	if periodAmount <= 0 || ttlAmount <= 0 {
		return Rule{}, fmt.Errorf("invalid rule %q (amounts must be > 0)", s)
	}

	period, err := unitToDuration(periodAmount, periodUnit)
	if err != nil {
		return Rule{}, fmt.Errorf("invalid period in %q: %w", s, err)
	}
	ttl, err := unitToDuration(ttlAmount, ttlUnit)
	if err != nil {
		return Rule{}, fmt.Errorf("invalid ttl in %q: %w", s, err)
	}

	if period > ttl {
		return Rule{}, fmt.Errorf("invalid rule %q (period > ttl)", s)
	}

	return Rule{Period: period, TTL: ttl, Raw: s}, nil
}

func unitToDuration(amount int, unit string) (time.Duration, error) {
	// month = 30 days, year = 365 days
	switch unit {
	case "s":
		return time.Duration(amount) * time.Second, nil
	case "min":
		return time.Duration(amount) * time.Minute, nil
	case "h":
		return time.Duration(amount) * time.Hour, nil
	case "d":
		return time.Duration(amount) * 24 * time.Hour, nil
	case "w":
		return time.Duration(amount) * 7 * 24 * time.Hour, nil
	case "m":
		return time.Duration(amount) * 30 * 24 * time.Hour, nil
	case "y":
		return time.Duration(amount) * 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit %q (use s|min|h|d|w|m|y)", unit)
	}
}
