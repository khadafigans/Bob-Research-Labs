package notify

import "testing"

func TestParseFilterMalformedFallsBackToMatchAll(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"", ""},
		{" JSON", `{not json`},
		{" JSON", `{"min_severity":`},
		{"", `{"min_severity": 123, "task_ids": "abc"}`},
		{"", `[1,2,3]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := ParseFilter([]byte(tc.raw))
			if f.MinSeverity != "" || len(f.TaskIDs) != 0 || len(f.AssetIDs) != 0 {
				t.Fatalf(" Filter， %+v", f)
			}
			ev := Snapshot{Kind: EventFindingCreated, Severity: "low", VulnClass: "XSS"}
			if !Match(f, ev) {
				t.Fatal(" Filter ")
			}
		})
	}
}

func TestMatchSeverityThreshold(t *testing.T) {
	ev := func(sev string) Snapshot {
		return Snapshot{Kind: EventFindingCreated, Severity: sev}
	}
	cases := []struct {
		min    string
		sev    string
		expect bool
	}{
		{"", "low", true},
		{"", "critical", true},
		{"high", "critical", true},
		{"high", "high", true},
		{"high", "medium", false},
		{"high", "low", false},
		{"critical", "high", false},
		{"critical", "critical", true},
		{"low", "", false},
		{"low", "unknown", false},
		{"", "", true},
	}
	for _, tc := range cases {
		got := Match(Filter{MinSeverity: tc.min}, ev(tc.sev))
		if got != tc.expect {
			t.Errorf("min=%q sev=%q:  %v  %v", tc.min, tc.sev, tc.expect, got)
		}
	}
}

func TestMatchScopeRestrictions(t *testing.T) {
	ev := Snapshot{
		Kind:      EventFindingCreated,
		Severity:  "high",
		TaskID:    7,
		AssetIDs:  []int64{10, 20},
		VulnClass: "SQL",
	}
	cases := []struct {
		name   string
		filter Filter
		expect bool
	}{
		{"=", Filter{}, true},
		{"", Filter{TaskIDs: []int64{7}}, true},
		{"", Filter{TaskIDs: []int64{8}}, false},
		{"", Filter{TaskIDs: []int64{8, 7}}, true},
		{"", Filter{AssetIDs: []int64{20, 99}}, true},
		{"", Filter{AssetIDs: []int64{99}}, false},
		{"", Filter{TaskIDs: []int64{7}, AssetIDs: []int64{10}}, true},
		{"", Filter{TaskIDs: []int64{7}, AssetIDs: []int64{99}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Match(tc.filter, ev); got != tc.expect {
				t.Errorf(" %v  %v", tc.expect, got)
			}
		})
	}
}

func TestMatchVulnClassKeywords(t *testing.T) {
	ev := func(class string) Snapshot {
		return Snapshot{Kind: EventFindingCreated, Severity: "high", VulnClass: class}
	}
	cases := []struct {
		name   string
		filter Filter
		class  string
		expect bool
	}{
		{"include =", Filter{}, "", true},
		{"include ", Filter{VulnClassInclude: []string{"SQL"}}, "SQL", true},
		{"include ", Filter{VulnClassInclude: []string{""}}, "SQL", false},
		{"include ", Filter{VulnClassInclude: []string{"", "SQL"}}, "SQL", true},
		{"", Filter{VulnClassInclude: []string{"sql"}}, "SQL", true},
		{"exclude ", Filter{VulnClassExclude: []string{""}}, "", false},
		{"exclude ", Filter{VulnClassExclude: []string{""}}, "SQL", true},
		{"", Filter{
			VulnClassInclude: []string{"SQL"},
			VulnClassExclude: []string{""},
		}, "SQL", false},
		{"", Filter{VulnClassInclude: []string{"", "  "}}, "SQL", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Match(tc.filter, ev(tc.class)); got != tc.expect {
				t.Errorf(" %v  %v", tc.expect, got)
			}
		})
	}
}

func TestMatchStatusChangeRequiresOptIn(t *testing.T) {
	ev := Snapshot{Kind: EventFindingStatusChanged, Severity: "critical", FromStatus: "pending", ToStatus: "fixed"}
	if Match(Filter{MinSeverity: "low"}, ev) {
		t.Fatal("")
	}
	if !Match(Filter{OnStatusChange: true}, ev) {
		t.Fatal(" on_status_change ")
	}
	//  on_status_change 。
	created := Snapshot{Kind: EventFindingCreated, Severity: "critical"}
	if !Match(Filter{MinSeverity: "low"}, created) {
		t.Fatal(" on_status_change")
	}
}
