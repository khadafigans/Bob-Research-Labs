package notify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func singleMsg() Message {
	return Message{
		Items: []Item{{
			FindingID: 42,
			Name:      ` "SQL" `,
			VulnClass: "SQL",
			Severity:  "high",
			Summary:   " id\n ",
			Assets:    []string{"a.example.com", "b.example.com"},
			DetailURL: "https://artex.local/function/findings/detail?id=42",
		}},
	}
}

func batchMsg(n int) Message {
	m := Message{Batch: true, WindowMinutes: 30, HomeURL: "https://artex.local/function/findings"}
	for i := 0; i < n; i++ {
		m.Items = append(m.Items, Item{
			FindingID: int64(i + 1),
			Name:      "" + itoa(i+1),
			VulnClass: "XSS",
			Severity:  "medium",
			Summary:   "",
			Assets:    []string{"target.example.com"},
		})
	}
	return m
}

func capturePost(t *testing.T, respBody string, assert func(t *testing.T, body map[string]any, r *http.Request)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Errorf(" JSON: %v\n: %s", err, raw)
			}
		}
		if assert != nil {
			assert(t, body, r)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, respBody)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDingTalkSendsActionCardWhenLinkPresent(t *testing.T) {
	srv := capturePost(t, `{"errcode":0,"errmsg":"ok"}`, func(t *testing.T, body map[string]any, _ *http.Request) {
		if body["msgtype"] != "actionCard" {
			t.Fatalf(" actionCard， %v", body["msgtype"])
		}
		card, _ := body["actionCard"].(map[string]any)
		if card["singleURL"] != "https://artex.local/function/findings/detail?id=42" {
			t.Errorf(": %v", card["singleURL"])
		}
	})
	if _, err := (dingTalkChannel{}).Send(context.Background(), map[string]any{"webhook": srv.URL}, singleMsg()); err != nil {
		t.Fatalf(": %v", err)
	}
}

func TestDingTalkFallsBackToMarkdownForBatch(t *testing.T) {
	srv := capturePost(t, `{"errcode":0,"errmsg":"ok"}`, func(t *testing.T, body map[string]any, _ *http.Request) {
		if body["msgtype"] != "markdown" {
			t.Fatalf(" markdown， %v", body["msgtype"])
		}
		md, _ := body["markdown"].(map[string]any)
		if !strings.Contains(md["text"].(string), "last 30 min") {
			t.Errorf(": %v", md["text"])
		}
	})
	if _, err := (dingTalkChannel{}).Send(context.Background(), map[string]any{"webhook": srv.URL}, batchMsg(3)); err != nil {
		t.Fatalf(": %v", err)
	}
}

// TestDingTalkBusinessErrorIsPermanent 「HTTP 200  errcode  0」。
func TestDingTalkBusinessErrorIsPermanent(t *testing.T) {
	srv := capturePost(t, `{"errcode":310000,"errmsg":"keywords not in content"}`, nil)
	_, err := (dingTalkChannel{}).Send(context.Background(), map[string]any{"webhook": srv.URL}, singleMsg())
	if err == nil {
		t.Fatal("errcode  0 ")
	}
	if !IsPermanent(err) {
		t.Fatalf("，， %v", err)
	}
	if !strings.Contains(err.Error(), "310000") {
		t.Errorf("， %v", err)
	}
}

func TestWeComTruncatesCJKWithinByteLimit(t *testing.T) {
	var contentLen int
	srv := capturePost(t, `{"errcode":0,"errmsg":"ok"}`, func(t *testing.T, body map[string]any, _ *http.Request) {
		md, _ := body["markdown"].(map[string]any)
		content, _ := md["content"].(string)
		contentLen = len(content)
		if !utf8.ValidString(content) {
			t.Fatal(" UTF-8——")
		}
	})
	m := batchMsg(200)
	if _, err := (weComChannel{}).Send(context.Background(), map[string]any{"webhook": srv.URL}, m); err != nil {
		t.Fatalf(": %v", err)
	}
	if contentLen > weComMarkdownLimit {
		t.Fatalf(" %d  %d", contentLen, weComMarkdownLimit)
	}
	if contentLen == 0 {
		t.Fatal("")
	}
}

func TestWeComRateLimitIsRetryableButKeyErrorIsPermanent(t *testing.T) {
	limited := capturePost(t, `{"errcode":45009,"errmsg":"api freq out of limit"}`, nil)
	_, err := (weComChannel{}).Send(context.Background(), map[string]any{"webhook": limited.URL}, singleMsg())
	if err == nil || IsPermanent(err) {
		t.Fatalf("45009 ，， %v", err)
	}

	badKey := capturePost(t, `{"errcode":93000,"errmsg":"invalid webhook url"}`, nil)
	_, err = (weComChannel{}).Send(context.Background(), map[string]any{"webhook": badKey.URL}, singleMsg())
	if err == nil || !IsPermanent(err) {
		t.Fatalf("93000  key ，，， %v", err)
	}
}

func TestFeishuCardStructureAndSign(t *testing.T) {
	const secret = "SECtest123"
	srv := capturePost(t, `{"code":0,"msg":"success"}`, func(t *testing.T, body map[string]any, _ *http.Request) {
		if body["msg_type"] != "interactive" {
			t.Fatalf("， %v", body["msg_type"])
		}
		card, _ := body["card"].(map[string]any)
		header, _ := card["header"].(map[string]any)
		if header["template"] != "orange" {
			t.Errorf("high  orange ， %v", header["template"])
		}
		if body["sign"] == nil || body["timestamp"] == nil {
			t.Fatalf(": %v", body)
		}
		elements, _ := card["elements"].([]any)
		foundButton := false
		for _, e := range elements {
			em, _ := e.(map[string]any)
			if em["tag"] != "action" {
				continue
			}
			actions, _ := em["actions"].([]any)
			for _, a := range actions {
				am, _ := a.(map[string]any)
				if am["url"] == "https://artex.local/function/findings/detail?id=42" {
					foundButton = true
				}
			}
		}
		if !foundButton {
			t.Fatal("")
		}
	})
	cfg := map[string]any{"webhook": srv.URL, "secret": secret}
	if _, err := (feishuChannel{}).Send(context.Background(), cfg, singleMsg()); err != nil {
		t.Fatalf(": %v", err)
	}
}

func TestFeishuWithoutSecretOmitsSign(t *testing.T) {
	srv := capturePost(t, `{"code":0,"msg":"success"}`, func(t *testing.T, body map[string]any, _ *http.Request) {
		if body["sign"] != nil || body["timestamp"] != nil {
			t.Fatalf(" secret : %v", body)
		}
	})
	if _, err := (feishuChannel{}).Send(context.Background(), map[string]any{"webhook": srv.URL}, singleMsg()); err != nil {
		t.Fatalf(": %v", err)
	}
}

func TestTelegramEscapesHTMLInUntrustedContent(t *testing.T) {
	var text string
	srv := capturePost(t, `{"ok":true}`, func(t *testing.T, body map[string]any, _ *http.Request) {
		text, _ = body["text"].(string)
		if body["parse_mode"] != "HTML" {
			t.Fatalf(" HTML ， %v", body["parse_mode"])
		}
	})
	m := Message{Items: []Item{{
		Severity: "high",
		Name:    `<script>alert(1)</script>`,
		Summary: "a & b < c",
	}}}
	if _, err := (telegramChannel{}).Send(context.Background(),
		map[string]any{"bot_token": "tok", "chat_id": "1", "base_url": srv.URL}, m); err != nil {
		t.Fatalf(": %v", err)
	}
	if strings.Contains(text, "<script>") {
		t.Fatalf(" HTML，: %q", text)
	}
	if !strings.Contains(text, "&lt;script&gt;") {
		t.Fatalf("， %q", text)
	}
	if !strings.Contains(text, "a &amp; b") {
		t.Fatalf("& ， %q", text)
	}
}

func TestTelegramErrorClassification(t *testing.T) {
	rateLimited := capturePost(t, `{"ok":false,"error_code":429,"description":"Too Many Requests"}`, nil)
	_, err := (telegramChannel{}).Send(context.Background(),
		map[string]any{"bot_token": "tok", "chat_id": "1", "base_url": rateLimited.URL}, singleMsg())
	if err == nil || IsPermanent(err) {
		t.Fatalf("429 ， %v", err)
	}

	forbidden := capturePost(t, `{"ok":false,"error_code":403,"description":"bot was blocked by the user"}`, nil)
	_, err = (telegramChannel{}).Send(context.Background(),
		map[string]any{"bot_token": "tok", "chat_id": "1", "base_url": forbidden.URL}, singleMsg())
	if err == nil || !IsPermanent(err) {
		t.Fatalf("403 ，， %v", err)
	}
}

func TestWebhookDefaultTemplateProducesValidJSON(t *testing.T) {
	// `"title": "{{.Title}}"`  JSON。{{json .}} 。
	srv := capturePost(t, `{"ok":true}`, func(t *testing.T, body map[string]any, _ *http.Request) {
		if body["title"] != `[🟠 High]  "SQL" ` {
			t.Errorf(": %v", body["title"])
		}
		items, _ := body["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("items  1， %d", len(items))
		}
		it, _ := items[0].(map[string]any)
		if it["summary"] != " id\n " {
			t.Errorf(": %v", it["summary"])
		}
		//  JSON （json:"...,string" ）。
		if _, ok := it["finding_id"].(float64); !ok {
			t.Errorf("finding_id ， %T", it["finding_id"])
		}
	})
	if _, err := (webhookChannel{}).Send(context.Background(), map[string]any{"url": srv.URL}, singleMsg()); err != nil {
		t.Fatalf(": %v", err)
	}
}

func TestWebhookCustomTemplateAndHeaders(t *testing.T) {
	srv := capturePost(t, `{"ok":true}`, func(t *testing.T, body map[string]any, r *http.Request) {
		if r.Header.Get("X-Token") != "s3cret" {
			t.Errorf(": %v", r.Header)
		}
		if body["msg"] != "3 " {
			t.Errorf(": %v", body["msg"])
		}
		if body["first"] != "1" {
			t.Errorf("range : %v", body["first"])
		}
	})
	cfg := map[string]any{
		"url":           srv.URL,
		"headers":       map[string]any{"X-Token": "s3cret"},
		"body_template": `{"msg": {{json (printf "%d " .Count)}}, "first": {{json (index .Items 0).Name}}}`,
	}
	if _, err := (webhookChannel{}).Send(context.Background(), cfg, batchMsg(3)); err != nil {
		t.Fatalf(": %v", err)
	}
}

func TestWebhookRejectsNonJSONRenderResult(t *testing.T) {
	cfg := map[string]any{"url": "https://example.com/hook", "body_template": `not json at all`}
	_, err := (webhookChannel{}).Send(context.Background(), cfg, singleMsg())
	if err == nil || !IsPermanent(err) {
		t.Fatalf(" JSON （，）， %v", err)
	}
}

func TestWebhookValidateCatchesBadConfigEarly(t *testing.T) {
	bad := []map[string]any{
		{},
		{"url": "file:///etc/passwd"},
		{"url": "https://example.com", "method": "DELETE"},
		{"url": "https://example.com", "body_template": `{{.Items.`},
	}
	for i, cfg := range bad {
		if err := (webhookChannel{}).Validate(cfg); err == nil {
			t.Errorf(" %d : %v", i, cfg)
		}
	}
}

func TestEmailMessageIsWellFormed(t *testing.T) {
	msg, err := buildEmailMessage("artex@example.com", []string{"a@example.com", "b@example.com"}, singleMsg())
	if err != nil {
		t.Fatalf(": %v", err)
	}
	if !strings.HasPrefix(msg, "From: artex@example.com\r\n") {
		t.Fatalf("From :\n%s", msg)
	}
	if !strings.Contains(msg, "To: a@example.com, b@example.com\r\n") {
		t.Fatalf("To :\n%s", msg)
	}
	if !strings.Contains(msg, "Subject: =?utf-8?") {
		t.Fatalf(" RFC 2047 :\n%s", msg)
	}
	if dec, err := new(mime.WordDecoder).DecodeHeader(mustExtractHeader(t, msg, "Subject")); err != nil {
		t.Fatalf(": %v", err)
	} else if !strings.Contains(dec, "SQL") {
		t.Fatalf(": %q", dec)
	}

	parts := strings.SplitN(msg, "\r\n\r\n", 2)
	if len(parts) != 2 {
		t.Fatal("/")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.TrimSpace(parts[1]), "\r\n", ""))
	if err != nil {
		t.Fatalf(" base64 : %v", err)
	}
	html := string(decoded)
	if !strings.HasPrefix(html, "<div") {
		t.Fatalf(" HTML: %.80s", html)
	}
	if !strings.Contains(html, `"SQL"`) {
		t.Fatalf(": %.200s", html)
	}
}

// TestEmailEscapesStructuralInjection ：
func TestEmailEscapesStructuralInjection(t *testing.T) {
	m := Message{
		Items: []Item{{
			Severity:  "high",
			Name:      `<script>alert(1)</script>`,
			Summary:   "a & b > c",
			DetailURL: `https://artex.local/x?a="onmouseover=alert(1)`,
		}},
	}
	html := htmlBody(m, 0)
	if strings.Contains(html, "<script>") {
		t.Fatalf("，: %s", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Fatalf(": %s", html)
	}
	if !strings.Contains(html, "a &amp; b &gt; c") {
		t.Fatalf("&  > : %s", html)
	}
	//  public_base_url，，
	if strings.Contains(html, `onmouseover=alert(1)">`) {
		t.Fatalf("href : %s", html)
	}
	if !strings.Contains(html, "&quot;") {
		t.Fatalf(": %s", html)
	}
}

func mustExtractHeader(t *testing.T, msg, name string) string {
	t.Helper()
	for _, line := range strings.Split(msg, "\r\n") {
		if strings.HasPrefix(line, name+": ") {
			return strings.TrimPrefix(line, name+": ")
		}
	}
	t.Fatalf(" %s ", name)
	return ""
}

func TestChannelValidateReportsMissingFields(t *testing.T) {
	cases := []struct {
		kind   string
		cfg    map[string]any
		substr string
	}{
		{KindDingTalk, map[string]any{}, "Webhook"},
		{KindFeishu, map[string]any{}, "Webhook"},
		{KindWeCom, map[string]any{}, "Webhook"},
		{KindTelegram, map[string]any{}, "Bot Token"},
		{KindTelegram, map[string]any{"bot_token": "t"}, "Chat ID"},
		{KindEmail, map[string]any{}, "SMTP"},
		{KindEmail, map[string]any{"host": "h"}, "port"},
		{KindEmail, map[string]any{"host": "h", "port": 587, "from": "f"}, "recipient"},
	}
	for _, tc := range cases {
		ch, ok := Get(tc.kind)
		if !ok {
			t.Fatalf(" %s ", tc.kind)
		}
		err := ch.Validate(tc.cfg)
		if err == nil {
			t.Errorf("%s  %v ", tc.kind, tc.cfg)
			continue
		}
		if !strings.Contains(err.Error(), tc.substr) {
			t.Errorf("%s  %q， %q", tc.kind, tc.substr, err.Error())
		}
	}
}

// TestEmailSMTPErrorClassification  SMTP 4xx/5xx 。
func TestEmailSMTPErrorClassification(t *testing.T) {
	cases := []struct {
		reply     string
		permanent bool
	}{
		{"450 4.7.1 Greylisting in action, please come back later", false},
		{"451 4.3.0 Temporary system failure", false},
		{"452 4.2.2 Mailbox full", false},
		{"550 5.1.1 User unknown", true},
		{"553 5.1.3 Bad address syntax", true},
		{"554 5.7.1 Relay access denied", true},
		{"unexpected EOF", false},
		{"", false},
	}
	for _, tc := range cases {
		err := smtpStageError("", errors.New(tc.reply))
		if got := IsPermanent(err); got != tc.permanent {
			t.Errorf(" %q:  permanent=%v  %v", tc.reply, tc.permanent, got)
		}
		if tc.reply != "" && !strings.Contains(err.Error(), tc.reply) {
			t.Errorf(" %q : %v", tc.reply, err)
		}
	}
}

func TestRegistryCoversAllKinds(t *testing.T) {
	want := []string{KindDingTalk, KindEmail, KindFeishu, KindTelegram, KindWebhook, KindWeCom}
	got := Kinds()
	if len(got) != len(want) {
		t.Fatalf(" %d， %d: %v", len(want), len(got), got)
	}
	for _, k := range want {
		if !ValidKind(k) {
			t.Errorf(" %s ", k)
		}
		if ch, ok := Get(k); !ok || ch.Kind() != k {
			t.Errorf(" %s  Kind() ", k)
		}
	}
	if ValidKind("nope") {
		t.Error("")
	}
}

func TestPermanentErrorUnwrap(t *testing.T) {
	base := &permanentSentinel{}
	err := Permanent(base)
	if !IsPermanent(err) {
		t.Fatal("")
	}
	if !strings.Contains(err.Error(), "sentinel") {
		t.Fatalf(": %v", err)
	}
	if Permanent(nil) != nil {
		t.Fatal("Permanent(nil)  nil")
	}
	if IsPermanent(nil) {
		t.Fatal("nil ")
	}
}

type permanentSentinel struct{}

func (*permanentSentinel) Error() string { return "sentinel" }
