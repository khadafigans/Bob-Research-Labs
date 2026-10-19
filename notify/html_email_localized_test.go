package notify

import (
	"strings"
	"testing"
)

// html.go   email.go      .
//  hasHan·hasHangul·assertKorean  notify_localized_test.go  ( ).
//

// TestHTMLItemLabelsLocalized         
func TestHTMLItemLabelsLocalized(t *testing.T) {
	m := Message{
		HomeURL: "https://example.com/panel",
		Items: []Item{{
			Name:       "sqli-login", // Title() = Name
			VulnClass:  "injection",  // Title()     
			Severity:   "high",       // SeverityLabel → ()
			Summary:    "login form is injectable",
			Assets:     []string{"host-a.example.com"},
			DetailURL:  "https://example.com/f/1",
			FromStatus: "pending", // IsStatusChange()=true → Status change 
			ToStatus:   "fixed",
		}},
	}
	out := htmlBody(m, 0)
	if hasHan(out) {
		t.Errorf("     :\n%s", out)
	}
	for _, want := range []string{
		"Status change", "Type", "Asset", "Summary", "View details", "View all on platform",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("  %q  is required:\n%s", want, out)
		}
	}
}

// TestHTMLBatchIntroLocalized  (digest)   ( )
func TestHTMLBatchIntroLocalized(t *testing.T) {
	items := []Item{{Name: "a", Severity: "high"}, {Name: "b", Severity: "low"}}

	withWindow := htmlBatchIntro(Message{Batch: true, WindowMinutes: 30, Items: items})
	assertKorean(t, "htmlBatchIntro()", withWindow)
	for _, want := range []string{"last 30 min", "new finding(s)", "2"} {
		if !strings.Contains(withWindow, want) {
			t.Errorf("  %q  is required: %q", want, withWindow)
		}
	}

	noWindow := htmlBatchIntro(Message{Batch: true, WindowMinutes: 0, Items: items})
	assertKorean(t, "htmlBatchIntro( )", noWindow)
	if !strings.Contains(noWindow, "2 new finding(s)") {
		t.Errorf("   %q   : %q", "2 new finding(s)", noWindow)
	}
	if strings.Contains(noWindow, "") {
		t.Errorf("  ''  : %q", noWindow)
	}
}

// TestEmailValidateLocalized  SMTP      ,
func TestEmailValidateLocalized(t *testing.T) {
	cases := []struct {
		name   string
		cfg    map[string]any
		substr string
	}{
		{"missing host", map[string]any{"port": float64(25), "from": "a@b.c", "to": []any{"d@e.f"}}, "SMTP"},
		{"bad port", map[string]any{"host": "h"}, "port"},
		{"missing from", map[string]any{"host": "h", "port": float64(25), "to": []any{"d@e.f"}}, "From"},
		{"missing to", map[string]any{"host": "h", "port": float64(25), "from": "a@b.c"}, "recipient"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := (emailChannel{}).Validate(tc.cfg)
			if err == nil {
				t.Fatalf("  : %v", tc.cfg)
			}
			assertKorean(t, "Validate("+tc.name+")", err.Error())
			if !strings.Contains(err.Error(), tc.substr) {
				t.Errorf("  %q   ,   %q", tc.substr, err.Error())
			}
		})
	}
}
