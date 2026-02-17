package retention

import (
	"testing"
	"time"
)

func TestBuildPrunePlan_OnlyIAB(t *testing.T) {
	now := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)

	names := []string{
		"IAB_20270101-101200",
		"IAB_20260205-110000",
		"IAB_20260204-110000",
		"manual_20260203",
		"IAB_not-a-time",
		"iab_20210101-135814",
		"IAB_20260204-11000",
	}

	plan, err := BuildPrunePlan(names, "1,1d2d", PruneOptions{
		Now:     now,
		DryRun:  true,
		Prefix:  IABSnapshotPrefix,
		ParseTS: ParseIABSnapshotTime,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plan.Unmanaged) != 4 {
		t.Fatalf("Unmanaged=%d want 4", len(plan.Unmanaged))
	}

	if len(plan.Future) != 1 {
		t.Fatalf("Future=%d want 1", len(plan.Future))
	}

	if len(plan.Keep) == 0 {
		t.Fatalf("expected some keeps")
	}
}
