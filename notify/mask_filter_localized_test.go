package notify

import (
	"errors"
	"strings"
	"testing"
)

// F4 ⑥: notify/mask.go·filter.go    (   · 
// server/notify_api.go  writeErr(400)     ) 
// . hasHan / hasHangul / assertKorean  notify_localized_test.go   .

// TestFilterValidateLocalized  min_severity       .
//   · (server/notify_api.go:264·369) 400  .
func TestFilterValidateLocalized(t *testing.T) {
	err := Filter{MinSeverity: "hgih"}.Validate()
	if err == nil {
		t.Fatal("    is required")
	}
	assertKorean(t, "Filter.Validate", err.Error())
	// low/medium/high/critical   enum     .
	for _, tok := range []string{"low", "medium", "high", "critical"} {
		if !strings.Contains(err.Error(), tok) {
			t.Errorf("  %q  : %q", tok, err.Error())
		}
	}
	if err := (Filter{MinSeverity: "high"}).Validate(); err != nil {
		t.Errorf("high    : %v", err)
	}
	if err := (Filter{}).Validate(); err != nil {
		t.Errorf("     : %v", err)
	}
}

// TestPrepareConfigUpdateUnknownKindLocalized      
//   (server/notify_api.go:326  400) .
func TestPrepareConfigUpdateUnknownKindLocalized(t *testing.T) {
	_, err := PrepareConfigUpdate("definitely-not-a-channel", map[string]any{}, map[string]any{})
	if err == nil {
		t.Fatal("    is required")
	}
	assertKorean(t, "PrepareConfigUpdate unknown kind", err.Error())
}

// TestDestinationChangedErrorLocalized    ErrDestinationChangedWithoutCredentials
//  : webhook   (url)     (headers)
func TestDestinationChangedErrorLocalized(t *testing.T) {
	_, err := PrepareConfigUpdate("webhook",
		map[string]any{
			"url":     "https://old.example.com/hook",
			"headers": map[string]any{"Authorization": "Bearer real-token"},
		},
		map[string]any{"url": "https://attacker.example.com/hook"},
	)
	if err == nil {
		t.Fatal("        is required")
	}
	var de *ErrDestinationChangedWithoutCredentials
	if !errors.As(err, &de) {
		t.Fatalf("   : %T", err)
	}
	assertKorean(t, "ErrDestinationChangedWithoutCredentials", err.Error())
	if !strings.Contains(err.Error(), "url") || !strings.Contains(err.Error(), "headers") {
		t.Errorf("   : %q", err.Error())
	}
}

// TestRejectMaskedInContainersLocalized        
// : headers(   )       PATCH .
func TestRejectMaskedInContainersLocalized(t *testing.T) {
	_, err := PrepareConfigUpdate("webhook",
		map[string]any{},
		map[string]any{"headers": map[string]any{"Authorization": MaskedPrefix + ":…abc123"}},
	)
	if err == nil {
		t.Fatal("       is required")
	}
	assertKorean(t, "rejectMaskedInContainers", err.Error())
	if !strings.Contains(err.Error(), MaskedPrefix) {
		t.Errorf("   is required: %q", err.Error())
	}
}
