package notify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

//
// http.Client.Do  *url.Error ** URL** ，
//
//	notification_deliveries.last_error  → 
//	GET /api/notify/deliveries      → ，
//

// credentialCases 「 URL 」：
// / query，，Telegram 。
var credentialCases = []struct {
	name   string
	ch     Channel
	cfg    map[string]any
	secret string
}{
	{
		name:   " access_token  query",
		ch:     dingTalkChannel{},
		cfg:    map[string]any{"webhook": "http://127.0.0.1:1/robot/send?access_token=" + leakProbeToken},
		secret: leakProbeToken,
	},
	{
		name:   " key  query",
		ch:     weComChannel{},
		cfg:    map[string]any{"webhook": "http://127.0.0.1:1/cgi-bin/webhook/send?key=" + leakProbeToken},
		secret: leakProbeToken,
	},
	{
		name:   " hook id ",
		ch:     feishuChannel{},
		cfg:    map[string]any{"webhook": "http://127.0.0.1:1/open-apis/bot/v2/hook/" + leakProbeToken},
		secret: leakProbeToken,
	},
	{
		name:   "Telegram bot token ",
		ch:     telegramChannel{},
		cfg:    map[string]any{"bot_token": leakProbeToken, "chat_id": "1", "base_url": "http://127.0.0.1:1"},
		secret: leakProbeToken,
	},
	{
		name:   "",
		ch:     dingTalkChannel{},
		cfg:    map[string]any{"webhook": "http://127.0.0.1:1/robot/send", "secret": leakProbeToken},
		secret: leakProbeToken,
	},
}

// leakProbeToken ，。
const leakProbeToken = "LEAKPROBE0123456789abcdef"

// TestChannelErrorsNeverLeakCredentials 。
func TestChannelErrorsNeverLeakCredentials(t *testing.T) {
	for _, tc := range credentialCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.ch.Send(context.Background(), tc.cfg, Message{
				Items: []Item{{FindingID: 1, Severity: "high", Name: ""}},
			})
			if err == nil {
				t.Fatal("")
			}
			assertNoSecret(t, err.Error(), tc.secret)
		})
	}
}

// TestChannelErrorsNeverLeakCredentialsInPermanentPath ：
func TestChannelErrorsNeverLeakCredentialsInPermanentPath(t *testing.T) {
	cases := []struct {
		name string
		ch   Channel
		cfg  map[string]any
	}{
		//  →  validateHTTPURL / url.Parse 。
		{"", dingTalkChannel{}, map[string]any{"webhook": "file:///" + leakProbeToken}},
		{"", weComChannel{}, map[string]any{"webhook": "gopher://" + leakProbeToken}},
		{"", feishuChannel{}, map[string]any{"webhook": "ftp://" + leakProbeToken + "/hook"}},
		{"Telegram API ", telegramChannel{}, map[string]any{"bot_token": "tok", "chat_id": "1", "base_url": "file://" + leakProbeToken}},
		{" Webhook ", webhookChannel{}, map[string]any{"url": "javascript:" + leakProbeToken}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.ch.Send(context.Background(), tc.cfg, Message{Items: []Item{{Severity: "high"}}})
			if err == nil {
				t.Fatal("")
			}
			assertNoSecret(t, err.Error(), leakProbeToken)
		})
	}
}

func assertNoSecret(t *testing.T, text, secret string) {
	t.Helper()
	if strings.Contains(text, secret) {
		t.Fatalf(" %q:\n    %s", secret, text)
	}
}

func TestRedactRequestTargetKeepsOnlySchemeAndHost(t *testing.T) {
	cases := map[string]string{
		"https://oapi.dingtalk.com/robot/send?access_token=S1":    "https://oapi.dingtalk.com/…",
		"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=S2": "https://qyapi.weixin.qq.com/…",
		"https://open.feishu.cn/open-apis/bot/v2/hook/S3":         "https://open.feishu.cn/…",
		"https://api.telegram.org/botS4/sendMessage":              "https://api.telegram.org/…",
		"http://10.0.0.5:8080/hook":                               "http://10.0.0.5:8080/…",
	}
	for in, want := range cases {
		got := redactRequestTarget(in)
		if got != want {
			t.Errorf("redactRequestTarget(%q) = %q， %q", in, got, want)
		}
		if parts := strings.SplitN(in, "://", 2); len(parts) == 2 {
			if hostAndRest := strings.SplitN(parts[1], "/", 2); len(hostAndRest) == 2 && hostAndRest[1] != "" {
				if strings.Contains(got, hostAndRest[1]) {
					t.Errorf("/ %q: %q", hostAndRest[1], got)
				}
			}
		}
	}
	for _, bad := range []string{"", "://", "not a url", "http://"} {
		if got := redactRequestTarget(bad); strings.Contains(got, bad) && bad != "" {
			t.Errorf(" %q  %q", bad, got)
		}
	}
}

// TestRedactTransportErrorStripsURL  *url.Error ：
//  http.Client.Do ，。
func TestRedactTransportErrorStripsURL(t *testing.T) {
	inner := errors.New("dial tcp 127.0.0.1:1: connect: connection refused")
	uerr := &url.Error{
		Op:  "Post",
		URL: "https://api.telegram.org/bot" + leakProbeToken + "/sendMessage",
		Err: inner,
	}
	got := redactTransportError(uerr)
	assertNoSecret(t, got, leakProbeToken)
	if !strings.Contains(got, "api.telegram.org") {
		t.Errorf(" host ， %q", got)
	}
	if !strings.Contains(got, "connection refused") {
		t.Errorf("， %q", got)
	}
	if !strings.Contains(got, "Post") {
		t.Errorf("， %q", got)
	}
}

// TestRedactURLsInTextHandlesFallback ： *url.Error 
func TestRedactURLsInTextHandlesFallback(t *testing.T) {
	in := fmt.Sprintf("（a.example → http://b.example/bot%s/send）", leakProbeToken)
	got := redactURLsInText(in)
	assertNoSecret(t, got, leakProbeToken)
	if !strings.Contains(got, "http://b.example/…") {
		t.Errorf("， %q", got)
	}
	if plain := "dial tcp: connection refused"; redactURLsInText(plain) != plain {
		t.Error("")
	}
}

// TestCrossHostRedirectRefused 「 URL  +  = 」。
// httptest  127.0.0.1 ， Host ，
func TestCrossHostRedirectRefused(t *testing.T) {
	var hit bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		_, _ = io.WriteString(w, `{"errcode":0,"errmsg":"ok"}`)
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/robot/send?access_token="+leakProbeToken, http.StatusTemporaryRedirect)
	}))
	defer redirector.Close()

	_, err := (dingTalkChannel{}).Send(context.Background(),
		map[string]any{"webhook": redirector.URL + "/robot/send?access_token=" + leakProbeToken},
		Message{Items: []Item{{Severity: "high"}}})
	if err == nil {
		t.Fatal("")
	}
	if hit {
		t.Fatal("——")
	}
	assertNoSecret(t, err.Error(), leakProbeToken)
}

// TestSameHostRedirectAllowed ：（），
func TestSameHostRedirectAllowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robot/send" {
			http.Redirect(w, r, "/robot/send/", http.StatusTemporaryRedirect)
			return
		}
		_, _ = io.WriteString(w, `{"errcode":0,"errmsg":"ok"}`)
	}))
	defer srv.Close()

	if _, err := (dingTalkChannel{}).Send(context.Background(),
		map[string]any{"webhook": srv.URL + "/robot/send"},
		Message{Items: []Item{{Severity: "high"}}}); err != nil {
		t.Fatalf(": %v", err)
	}
}
