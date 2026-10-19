package report

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/Autumn-27/artex/db"
)

// containsHan   CJK ()   true  .
// (unicode.Hangul)···    .
func containsHan(s string) (rune, bool) {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return r, true
		}
	}
	return 0, false
}

func sampleFindingNode(t *testing.T) *db.Node {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"vulnclass": "SQL Injection",
		"name":      "  SQL ",
		"severity":  "high",
		"summary":   "    SQL  .",
		"evidence":  map[string]string{"poc": "' OR '1'='1"},
	})
	if err != nil {
		t.Fatalf("payload  : %v", err)
	}
	return &db.Node{ID: 1, Kind: "finding", Payload: payload}
}

func sampleDBFinding() *db.DBFinding {
	return &db.DBFinding{
		ID:              123,
		VulnClass:       "SQL Injection",
		Name:            "  SQL ",
		Severity:        "high",
		Summary:         "    SQL  .",
		Evidence:        "' OR '1'='1",
		Status:          "confirmed",
		Report:          "  .",
		CreatedAt:       time.Date(2026, 10, 4, 2, 30, 0, 0, time.UTC),
		EvidenceVersion: 2,
		TaskDescription: "   ",
		TrafficBindings: []db.FindingTrafficBinding{
			{
				ID:   9,
				Role: "request",
				Note: "   ",
				Snapshot: db.TrafficEvidenceSnapshot{
					Method: "POST",
					URL:    "https://sandbox.local/login",
					Status: 200,
				},
			},
		},
	}
}

// TestReportMarkdownLocalized    (report.Markdown)  
func TestReportMarkdownLocalized(t *testing.T) {
	out := Markdown(Input{
		Title:       " ",
		Goal:        "  ",
		GeneratedAt: time.Date(2026, 10, 4, 2, 30, 0, 0, time.UTC),
		AssetCounts: map[string]int{"host": 2, "url": 5},
		Findings:    []*db.Node{sampleFindingNode(t)},
	})
	if r, ok := containsHan(out); ok {
		t.Fatalf("    %q   :\n%s", string(r), out)
	}
	for _, want := range []string{
		"#   :  ",
		"- ** **:   ",
		"- ** **:",
		"## ",
		"-  : **1**",
		"## ",
		"**PoC / :**",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("  %q  ", want)
		}
	}
}

// TestReportMarkdownEmptyLocalized        .
func TestReportMarkdownEmptyLocalized(t *testing.T) {
	out := Markdown(Input{Title: " ", GeneratedAt: time.Now()})
	if r, ok := containsHan(out); ok {
		t.Fatalf("    %q   :\n%s", string(r), out)
	}
	if !strings.Contains(out, "_   ._") {
		t.Errorf("     :\n%s", out)
	}
}

// TestFindingsMarkdownLocalized   / Markdown    
func TestFindingsMarkdownLocalized(t *testing.T) {
	f := sampleDBFinding()
	out := FindingsMarkdown([]*db.DBFinding{f}, time.Date(2026, 10, 4, 2, 30, 0, 0, time.UTC))
	if r, ok := containsHan(out); ok {
		t.Fatalf("     %q   :\n%s", string(r), out)
	}
	for _, want := range []string{
		"#   ",
		"- ** **:",
		"- ** **: 1",
		"## ",
		"|  |  |",
		"|  |", "|  |", "|  |", "|  |", //  (UI status.severity  )
		"##  ",
		"- ****: SQL Injection",
		"- ****: confirmed",
		"- ** **:    ",
		"- ** **:",
		"**:**",
		"** :**",
		"##   ",
		" : 2,  : 1.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("  %q  ", want)
		}
	}

	single := SingleFindingMarkdown(f, time.Date(2026, 10, 4, 2, 30, 0, 0, time.UTC))
	if r, ok := containsHan(single); ok {
		t.Fatalf("    %q   :\n%s", string(r), single)
	}
	for _, want := range []string{
		"- ****: high",
		"## ",
		"## ",
		"##  ",
		"[ ](evidence/123/9/request.http)",
		"[ ](evidence/123/9/response.http)",
	} {
		if !strings.Contains(single, want) {
			t.Errorf("  %q  ", want)
		}
	}
}

// TestFindingsCSVLocalized  CSV       .
func TestFindingsCSVLocalized(t *testing.T) {
	raw := FindingsCSV([]*db.DBFinding{sampleDBFinding()})
	out := strings.TrimPrefix(string(raw), "\xEF\xBB\xBF")
	header := strings.SplitN(out, "\n", 2)[0]
	if r, ok := containsHan(header); ok {
		t.Fatalf("CSV    %q   : %s", string(r), header)
	}
	for _, col := range []string{"ID", "", "", "", "", " ", " ", "", "  ", "  ID"} {
		if !strings.Contains(header, col) {
			t.Errorf("CSV  %q  : %s", col, header)
		}
	}
}
