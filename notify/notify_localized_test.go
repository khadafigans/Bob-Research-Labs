package notify

import (
	"strings"
	"testing"
)

// hasHan reports whether s contains any CJK Unified Ideograph.
func hasHan(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// hasHangul reports whether s contains any Hangul syllable.
func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7AF {
			return true
		}
	}
	return false
}

func assertEnglishLabel(t *testing.T, where, got string) {
	t.Helper()
	if hasHan(got) {
		t.Errorf("%s: unexpected CJK han remaining: %q", where, got)
	}
	if hasHangul(got) {
		t.Errorf("%s: unexpected Hangul remaining: %q", where, got)
	}
}

// assertKorean kept as name for older localized tests; now asserts English (no Hangul/Han).
func assertKorean(t *testing.T, where, got string) {
	assertEnglishLabel(t, where, got)
}

func TestSeverityLabelLocalized(t *testing.T) {
	want := map[string]string{
		"critical": "Critical",
		"high":     "High",
		"medium":   "Medium",
		"low":      "Low",
	}
	for sev, label := range want {
		got := SeverityLabel(sev)
		assertEnglishLabel(t, "SeverityLabel/"+sev, got)
		if !strings.Contains(got, label) {
			t.Errorf("SeverityLabel(%q)=%q, want substring %q", sev, got, label)
		}
	}
	if got := SeverityLabel("unknown-sev"); got != "unknown-sev" {
		t.Errorf("unknown severity should pass through, got %q", got)
	}
}

func TestStatusLabelLocalized(t *testing.T) {
	want := map[string]string{
		"pending":        "Pending",
		"in_progress":    "In progress",
		"confirmed":      "Confirmed",
		"resolved":       "Resolved",
		"fixed":          "Fixed",
		"false_positive": "False positive",
		"ignored":        "Ignored",
		"duplicate":      "Duplicate",
		"risk_accepted":  "Risk accepted",
	}
	for status, label := range want {
		got := StatusLabel(status)
		assertEnglishLabel(t, "StatusLabel/"+status, got)
		if got != label {
			t.Errorf("StatusLabel(%q)=%q, want %q", status, got, label)
		}
	}
	if got := StatusLabel("weird"); got != "weird" {
		t.Errorf("unknown status should pass through, got %q", got)
	}
}

func TestItemTitlePlaceholderLocalized(t *testing.T) {
	if got := (Item{}).Title(); got != "(unnamed finding)" {
		t.Errorf("empty item title = %q, want (unnamed finding)", got)
	}
	assertEnglishLabel(t, "Title/empty", (Item{}).Title())
	if n := (Item{Name: "Login SQLi"}).Title(); n != "Login SQLi" {
		t.Errorf("name priority broken, got %q", n)
	}
	if v := (Item{VulnClass: "xss"}).Title(); v != "xss" {
		t.Errorf("vulnclass fallback broken, got %q", v)
	}
}

func TestAssetLineLocalized(t *testing.T) {
	got := assetLine([]string{"a.example.com", "b.example.com"}, 5)
	if !strings.Contains(got, ", ") {
		t.Errorf("expected ASCII comma separator, got %q", got)
	}
	if hasHan(got) || hasHangul(got) {
		t.Errorf("asset line has CJK/Hangul: %q", got)
	}
	got = assetLine([]string{"a", "b", "c", "d", "e"}, 3)
	if !strings.Contains(got, "(5 total)") {
		t.Errorf("want total count as (5 total), got %q", got)
	}
}
