package db

import (
	"testing"
	"unicode"
)

// assertRetestReasonKorean         
// (F9). FinishFindingRetest·RecoverFindingRetests  SQL   
func assertRetestReasonKorean(t *testing.T, name, s string) {
	t.Helper()
	if s == "" {
		t.Fatalf("%s:  ", name)
	}
	hasHangul := false
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			t.Fatalf("%s:    : %q", name, s)
		}
		if unicode.Is(unicode.Hangul, r) {
			hasHangul = true
		}
	}
	if !hasHangul {
		t.Fatalf("%s:  : %q", name, s)
	}
}

// TestFindingRetestReasonsLocalized  finding_retests.error    
// (finding-retest-panel)  item.error        .
// server/conversations.go   (convRetest*)  ·,   
func TestFindingRetestReasonsLocalized(t *testing.T) {
	assertRetestReasonKorean(t, "retestNoConclusionReason", retestNoConclusionReason)
	assertRetestReasonKorean(t, "retestServiceRestartReason", retestServiceRestartReason)
}
