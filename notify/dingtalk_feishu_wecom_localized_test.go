package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// asciiStatusChangeItem   (ASCII) Status change  .
func asciiStatusChangeItem() Item {
	return Item{
		FindingID:  7,
		Name:       "login-flaw",
		VulnClass:  "SQLi",
		Severity:   "high",
		Summary:    "SQL injection via q param",
		Assets:     []string{"a.example.com"},
		DetailURL:  "https://platform.example/finding/7",
		FromStatus: "pending",
		ToStatus:   "fixed",
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("  : %v", err)
	}
	return string(b)
}

// TestFeishuItemLinesLocalized       (Status change··
// ·)  markdown.go(F4②)   . feishu.go 
func TestFeishuItemLinesLocalized(t *testing.T) {
	got := feishuItemLines(asciiStatusChangeItem())
	assertKorean(t, "feishuItemLines", got)
	for _, want := range []string{
		"**Status change**: Pending → Fixed",
		"**Type**: SQLi",
		"**Asset**: a.example.com",
		"**Summary**: SQL injection via q param",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("   %q   :\n%s", want, got)
		}
	}
}

// TestFeishuCardButtonsLocalized     (View details·View all on platform)
func TestFeishuCardButtonsLocalized(t *testing.T) {
	// : "View details" (DetailURL  )
	single, _ := feishuCard(Message{Items: []Item{asciiStatusChangeItem()}})
	singleJSON := mustJSON(t, single)
	if hasHan(singleJSON) {
		t.Errorf("     :\n%s", singleJSON)
	}
	if !strings.Contains(singleJSON, "View details") {
		t.Errorf("   'View details'   :\n%s", singleJSON)
	}

	// : "View all on platform" (HomeURL  )
	batch, _ := feishuCard(Message{Batch: true, HomeURL: "https://platform.example", Items: hanFreeItems(2)})
	batchJSON := mustJSON(t, batch)
	if hasHan(batchJSON) {
		t.Errorf("     :\n%s", batchJSON)
	}
	if !strings.Contains(batchJSON, "View all on platform") {
		t.Errorf("   'View all on platform'   :\n%s", batchJSON)
	}
}

// TestDingTalkActionCardButtonLocalized   ActionCard  "View details" (singleTitle)
func TestDingTalkActionCardButtonLocalized(t *testing.T) {
	var singleTitle string
	srv := capturePost(t, `{"errcode":0,"errmsg":"ok"}`, func(_ *testing.T, body map[string]any, _ *http.Request) {
		card, _ := body["actionCard"].(map[string]any)
		singleTitle, _ = card["singleTitle"].(string)
	})
	m := Message{Items: []Item{asciiStatusChangeItem()}}
	if _, err := (dingTalkChannel{}).Send(context.Background(), map[string]any{"webhook": srv.URL}, m); err != nil {
		t.Fatalf(" : %v", err)
	}
	if singleTitle != "View details" {
		t.Errorf(" ActionCard singleTitle  'View details'  ,   %q", singleTitle)
	}
}

// TestChinaPlatformValidateLocalized  ··    
func TestChinaPlatformValidateLocalized(t *testing.T) {
	for _, kind := range []string{KindDingTalk, KindFeishu, KindWeCom} {
		ch, ok := Get(kind)
		if !ok {
			t.Fatalf("%s    ", kind)
		}

		//  : "Webhook URL is required"
		missing := ch.Validate(map[string]any{})
		if missing == nil {
			t.Fatalf("%s:     ", kind)
		}
		assertKorean(t, kind+" missing", missing.Error())
		if !strings.Contains(missing.Error(), "Webhook") || !strings.Contains(missing.Error(), "is required") {
			t.Errorf("%s:   'Webhook URL is required'  ,   %q", kind, missing)
		}

		//  (ftp): "Invalid Webhook URL: …"
		bad := ch.Validate(map[string]any{"webhook": "ftp://x"})
		if bad == nil {
			t.Fatalf("%s: ftp    ", kind)
		}
		if hasHan(bad.Error()) {
			t.Errorf("%s:      : %q", kind, bad)
		}
		if !strings.Contains(bad.Error(), "Invalid Webhook URL") {
			t.Errorf("%s:    'Invalid Webhook URL'   ,   %q", kind, bad)
		}
	}
}

// TestChinaPlatformSendErrorsLocalized  ··    
//  (last_error)  .   UI  (DingTalk/Feishu/
// WeCom) .  errmsg  ASCII      
func TestChinaPlatformSendErrorsLocalized(t *testing.T) {
	//   (HTTP 200 + body  errcode/code != 0) .
	bizCases := []struct {
		kind     string
		resp     string
		wantSubs []string
	}{
		{KindDingTalk, `{"errcode":310000,"errmsg":"keyword not matched"}`, []string{"DingTalk error", "310000"}},
		{KindFeishu, `{"code":19021,"msg":"sign error"}`, []string{"Feishu error", "19021"}},
		{KindWeCom, `{"errcode":45009,"errmsg":"freq out of limit"}`, []string{"WeCom rate limited", "45009"}},
		{KindWeCom, `{"errcode":93000,"errmsg":"invalid webhook"}`, []string{"WeCom error", "93000"}},
	}
	for _, tc := range bizCases {
		srv := capturePost(t, tc.resp, nil)
		ch, ok := Get(tc.kind)
		if !ok {
			t.Fatalf("%s    ", tc.kind)
		}
		_, err := ch.Send(context.Background(), map[string]any{"webhook": srv.URL}, singleMsg())
		if err == nil {
			t.Fatalf("%s: %s   ", tc.kind, tc.resp)
		}
		if hasHan(err.Error()) {
			t.Errorf("%s:     : %q", tc.kind, err)
		}
		for _, sub := range tc.wantSubs {
			if !strings.Contains(err.Error(), sub) {
				t.Errorf("%s:   %q   ,   %q", tc.kind, sub, err)
			}
		}
	}

	//  JSON      : "<> Failed to parse".
	parseCases := []struct {
		kind     string
		platform string
	}{
		{KindDingTalk, "DingTalk"},
		{KindFeishu, "Feishu"},
		{KindWeCom, "WeCom"},
	}
	for _, pc := range parseCases {
		srv := capturePost(t, `not-json`, nil)
		ch, ok := Get(pc.kind)
		if !ok {
			t.Fatalf("%s    ", pc.kind)
		}
		_, err := ch.Send(context.Background(), map[string]any{"webhook": srv.URL}, singleMsg())
		if err == nil {
			t.Fatalf("%s: JSON     ", pc.kind)
		}
		want := "Failed to parse " + pc.platform
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%s:    %q   ,   %q", pc.kind, want, err)
		}
	}
}
