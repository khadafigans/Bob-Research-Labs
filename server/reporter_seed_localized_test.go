package server

import (
	"strings"
	"testing"
	"unicode"
)

func TestReporterSeedLabelsLocalized(t *testing.T) {
	for _, tc := range []struct {
		label, got string
	}{
		{"reporter_name", reporterAgentName},
		{"reporter_description", reporterAgentDescription},
	} {
		if tc.got == "" {
			t.Fatalf("%s empty", tc.label)
		}
		for _, r := range tc.got {
			if unicode.In(r, unicode.Hangul) || (r >= 0x4E00 && r <= 0x9FFF) {
				t.Fatalf("%s has CJK/Hangul: %q", tc.label, tc.got)
			}
		}
	}
	if !strings.Contains(reporterAgentName, "Report Writer") {
		t.Fatalf("reporter name = %q, want Report Writer", reporterAgentName)
	}
	if !strings.Contains(strings.ToLower(reporterToolCallMessage), "report_finding") {
		t.Fatalf("reporter tool-call message should mention report_finding")
	}
	for _, r := range reporterToolCallMessage {
		if unicode.In(r, unicode.Hangul) || (r >= 0x4E00 && r <= 0x9FFF) {
			t.Fatalf("reporterToolCallMessage still has CJK/Hangul: %q", reporterToolCallMessage)
		}
	}
}
