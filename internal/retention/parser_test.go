package retention

import "testing"

func TestParseSchedule_OK(t *testing.T) {
	s, err := ParseSchedule("6,1h2d,1d2w")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.AlwaysKeep != 6 {
		t.Fatalf("AlwaysKeep=%d want 6", s.AlwaysKeep)
	}
	if len(s.Rules) != 2 {
		t.Fatalf("Rules=%d want 2", len(s.Rules))
	}
}

func TestParseSchedule_BadRule(t *testing.T) {
	_, err := ParseSchedule("1x2d")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseSchedule_InvalidPolicies(t *testing.T) {
	tests := []struct {
		name   string
		policy string
	}{
		{name: "always keep twice", policy: "3,5"},
		{name: "always keep not first", policy: "1d2w,3"},
		{name: "duplicate period", policy: "1d2w,1d1w"},
		{name: "period > ttl", policy: "2d1d"},
		{name: "unknown period unit", policy: "1x1d"},
		{name: "unknown ttl unit", policy: "1d1x"},
		{name: "zero period amount", policy: "0d1w"},
		{name: "zero ttl amount", policy: "1d0w"},
		{name: "garbage", policy: "abc"},
		{name: "empty element", policy: "3,,1d2w"},
		{name: "first element empty", policy: ",,2"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseSchedule(tc.policy)
			if err == nil {
				t.Fatalf("expected error for policy %q", tc.policy)
			}
		})
	}
}
