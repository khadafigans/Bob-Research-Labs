package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

//   F4 ④⑤(telegram·webhook· HTTP  )  
// . telegram.go   · , webhook.go    ,
// http.go   ,     validateHTTPURL  
//  .  hasHan·hasHangul·assertKorean 
// notify_localized_test.go  ( ).
//

// TestTelegramItemLabelsLocalized   Telegram      
func TestTelegramItemLabelsLocalized(t *testing.T) {
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
	out, kept := telegramHTML(m)
	if kept != 1 {
		t.Fatalf("  1   ,   %d", kept)
	}
	if hasHan(out) {
		t.Errorf("Telegram     :\n%s", out)
	}
	for _, want := range []string{"Status change", "Type", "Asset", "Summary", "View details"} {
		if !strings.Contains(out, want) {
			t.Errorf("Telegram body missing label %q:\n%s", want, out)
		}
	}
}

// TestTelegramBatchTitleLocalized  (digest)  · 
//   . markdownTitle      
func TestTelegramBatchTitleLocalized(t *testing.T) {
	items := []Item{{Name: "a", Severity: "high"}, {Name: "b", Severity: "low"}}

	//   + : ·· N··next message· N.
	over := telegramBatchTitle(Message{WindowMinutes: 30}, items, 5)
	assertKorean(t, "telegramBatchTitle()", over)
	for _, want := range []string{"Vulnerability digest · 5 total", "showing first 2", "remaining 3", "next message", "Last 30 min"} {
		if !strings.Contains(over, want) {
			t.Errorf("  %q  is required: %q", want, over)
		}
	}

	full := telegramBatchTitle(Message{}, items, 2)
	assertKorean(t, "telegramBatchTitle()", full)
	if !strings.Contains(full, "Vulnerability digest · 2 total") {
		t.Errorf("  'Vulnerability digest · 2 total'   : %q", full)
	}
	if strings.Contains(full, "next message") {
		t.Errorf("  'next message'  : %q", full)
	}
}

// TestTelegramValidateLocalized        
//    .   Bot Token·Chat ID   .
func TestTelegramValidateLocalized(t *testing.T) {
	if err := (telegramChannel{}).Validate(map[string]any{}); err == nil {
		t.Fatal("bot_token    ")
	} else {
		assertKorean(t, "telegram Validate(bot_token)", err.Error())
		if !strings.Contains(err.Error(), "Bot Token") {
			t.Errorf("Bot Token   : %q", err.Error())
		}
	}
	if err := (telegramChannel{}).Validate(map[string]any{"bot_token": "t"}); err == nil {
		t.Fatal("chat_id    ")
	} else {
		assertKorean(t, "telegram Validate(chat_id)", err.Error())
		if !strings.Contains(err.Error(), "Chat ID") {
			t.Errorf("Chat ID   : %q", err.Error())
		}
	}
}

// TestWebhookValidateLocalized   Webhook      
func TestWebhookValidateLocalized(t *testing.T) {
	if err := (webhookChannel{}).Validate(map[string]any{}); err == nil {
		t.Fatal("url    ")
	} else {
		assertKorean(t, "webhook Validate(url)", err.Error())
		if !strings.Contains(err.Error(), "Target URL") {
			t.Errorf(" URL   : %q", err.Error())
		}
	}
	if err := (webhookChannel{}).Validate(map[string]any{"url": "https://example.com/hook", "method": "DELETE"}); err == nil {
		t.Fatal("DELETE    ")
	} else {
		assertKorean(t, "webhook Validate(method)", err.Error())
		if !strings.Contains(err.Error(), "GET/POST/PUT/PATCH") {
			t.Errorf("    : %q", err.Error())
		}
	}
	if err := (webhookChannel{}).Validate(map[string]any{"url": "https://example.com/hook", "body_template": "{{"}); err == nil {
		t.Fatal("    ")
	} else {
		assertKorean(t, "webhook Validate(template)", err.Error())
		if !strings.Contains(err.Error(), "template") {
			t.Errorf("   : %q", err.Error())
		}
	}
}

// TestValidateHTTPURLLocalized   (telegram·webhook·dingtalk·feishu·wecom)
func TestValidateHTTPURLLocalized(t *testing.T) {
	if err := validateHTTPURL("ftp://example.com/x"); err == nil {
		t.Fatal("ftp   ")
	} else {
		assertKorean(t, "validateHTTPURL(scheme)", err.Error())
		if !strings.Contains(err.Error(), "http") {
			t.Errorf(" (http/https)  : %q", err.Error())
		}
	}
	if err := validateHTTPURL("http://"); err == nil {
		t.Fatal("    ")
	} else {
		assertKorean(t, "validateHTTPURL(host)", err.Error())
	}
}

// TestHTTPLocalTargetBlockedLocalized  /     
func TestHTTPLocalTargetBlockedLocalized(t *testing.T) {
	t.Setenv(AllowLocalTargetsEnv, "") //     
	err := blockInternalDial("tcp", "127.0.0.1:25", nil)
	if err == nil {
		t.Fatal("    ")
	}
	assertKorean(t, "blockInternalDial", err.Error())
	if !strings.Contains(err.Error(), AllowLocalTargetsEnv) {
		t.Errorf("   (%s)  : %q", AllowLocalTargetsEnv, err.Error())
	}
}

// TestHTTPRedactHelpersLocalized   (redact)   
func TestHTTPRedactHelpersLocalized(t *testing.T) {
	if got := redactRequestTarget(""); !strings.Contains(got, "unresolvable") {
		t.Errorf("     ,   %q", got)
	}
	// *url.Error  Err  nil  .
	got := redactTransportError(&url.Error{Op: "Get", URL: "http://api.example", Err: nil})
	if hasHan(got) {
		t.Errorf("     : %q", got)
	}
	if !strings.Contains(got, "unknown error") {
		t.Errorf("Err  nil  '   '  ,   %q", got)
	}
}

// TestHTTPStatusErrorsLocalized   doJSON   (/
func TestHTTPStatusErrorsLocalized(t *testing.T) {
	t.Setenv(AllowLocalTargetsEnv, "1")
	cases := []struct {
		code int
		want string
	}{
		{http.StatusForbidden, "rejected the request"},
		{http.StatusInternalServerError, "Remote server error"},
		{http.StatusTooManyRequests, "rate-limited"},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.code)
			_, _ = w.Write([]byte("body"))
		}))
		_, err := doJSON(context.Background(), http.MethodPost, srv.URL, nil, map[string]any{"a": 1})
		srv.Close()
		if err == nil {
			t.Fatalf("HTTP %d   ", tc.code)
		}
		assertKorean(t, "doJSON(HTTP status)", err.Error())
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("HTTP %d  %q   ,   %q", tc.code, tc.want, err.Error())
		}
	}
}
