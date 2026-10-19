package notify

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

//
// 。 TestMain  AllowLocalTargetsEnv， SSRF 

func TestMain(m *testing.M) {
	_ = os.Setenv(AllowLocalTargetsEnv, "1")
	os.Exit(m.Run())
}

// TestDialGuardRejectsLoopbackByDefault  SSRF ：
func TestDialGuardRejectsLoopbackByDefault(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		_, _ = io.WriteString(w, `{"errcode":0}`)
	}))
	defer srv.Close()

	t.Setenv(AllowLocalTargetsEnv, "") //  = 
	_, err := (dingTalkChannel{}).Send(context.Background(),
		map[string]any{"webhook": srv.URL + "/robot/send"}, Message{Items: []Item{{Severity: "high"}}})
	if err == nil {
		t.Fatal("")
	}
	if hit {
		t.Fatal("——")
	}
	if !strings.Contains(err.Error(), AllowLocalTargetsEnv) {
		t.Errorf(": %v", err)
	}
}

// TestDialGuardAllowsLoopbackWhenOptedIn ：，
func TestDialGuardAllowsLoopbackWhenOptedIn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"errcode":0,"errmsg":"ok"}`)
	}))
	defer srv.Close()

	t.Setenv(AllowLocalTargetsEnv, "1")
	if _, err := (dingTalkChannel{}).Send(context.Background(),
		map[string]any{"webhook": srv.URL + "/robot/send"}, Message{Items: []Item{{Severity: "high"}}}); err != nil {
		t.Fatalf(": %v", err)
	}
}

func TestIsBlockedDialIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "127.1.2.3", "::1",
		"169.254.169.254", // ——
		"169.254.1.1", "fe80::1",
		"0.0.0.0", "::",
		"224.0.0.1", "ff02::1",
		"::ffff:127.0.0.1", // IPv4-mapped ，
		"",
	}
	for _, s := range blocked {
		if !isBlockedDialIP(net.ParseIP(s)) {
			t.Errorf("%s ", s)
		}
	}
	// RFC1918 ****： Mattermost / SMTP 。
	allowed := []string{"10.0.0.5", "172.16.3.4", "192.168.1.10", "8.8.8.8", "2606:4700::1111"}
	for _, s := range allowed {
		if isBlockedDialIP(net.ParseIP(s)) {
			t.Errorf("%s （）", s)
		}
	}
}

// TestValidateHTTPURLRejectsLiteralPrivateTargets ：
func TestValidateHTTPURLRejectsLiteralPrivateTargets(t *testing.T) {
	t.Setenv(AllowLocalTargetsEnv, "")
	for _, raw := range []string{
		"http://127.0.0.1:8080/hook",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::1]:8080/hook",
	} {
		if err := validateHTTPURL(raw); err == nil {
			t.Errorf("%s ", raw)
		}
	}
	for _, raw := range []string{"https://oapi.dingtalk.com/robot/send", "http://10.0.0.9/hook"} {
		if err := validateHTTPURL(raw); err != nil {
			t.Errorf("%s : %v", raw, err)
		}
	}
}

// TestValidateHTTPURLErrorNeverLeaksCredentials 。
//
// url.Parse **** *url.Error， Error() 。
//  http.Client.Do ，；「」
// （file://、gopher://、ftp://） url.Parse 、 scheme ，
func TestValidateHTTPURLErrorNeverLeaksCredentials(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/%zz?access_token=" + leakProbeToken,         // 
		"https://a.example.com:port/x?access_token=" + leakProbeToken, // 
		"http://[::1?access_token=" + leakProbeToken,                  // 
	}
	for _, raw := range cases {
		if _, err := url.Parse(raw); err == nil {
			t.Errorf("%q ，", raw)
			continue
		}
		err := validateHTTPURL(raw)
		if err == nil {
			t.Errorf("%q ", raw)
			continue
		}
		assertNoSecret(t, err.Error(), leakProbeToken)
	}
	t.Setenv(AllowLocalTargetsEnv, "")
	err := (dingTalkChannel{}).Validate(map[string]any{"webhook": cases[0]})
	if err == nil {
		t.Fatal("")
	}
	assertNoSecret(t, err.Error(), leakProbeToken)
}

// TestEmailDialGuardRejectsLoopbackByDefault  SMTP 。
//
//  net.Dialer， SSRF ：host 
// 169.254.169.254  127.0.0.1 ， smtp.NewClient 
//
//  TestMain  AllowLocalTargetsEnv（ 127.0.0.1 
func TestEmailDialGuardRejectsLoopbackByDefault(t *testing.T) {
	f := newFakeSMTP(t)
	cfg := emailCfg(t, f, nil)

	t.Setenv(AllowLocalTargetsEnv, "") //  = 
	_, err := (emailChannel{}).Send(context.Background(), cfg, singleMsg())
	if err == nil {
		t.Fatal("")
	}
	if f.sawCommand("EHLO") || f.sawCommand("HELO") {
		t.Fatal("SMTP ——")
	}
	if !strings.Contains(err.Error(), AllowLocalTargetsEnv) {
		t.Errorf(": %v", err)
	}
}

// TestEmailDialGuardAllowsLoopbackWhenOptedIn ：
func TestEmailDialGuardAllowsLoopbackWhenOptedIn(t *testing.T) {
	f := newFakeSMTP(t)
	cfg := emailCfg(t, f, nil)

	t.Setenv(AllowLocalTargetsEnv, "1")
	if _, err := (emailChannel{}).Send(context.Background(), cfg, singleMsg()); err != nil {
		t.Fatalf(" SMTP : %v", err)
	}
	if !f.sawCommand("EHLO") {
		t.Fatal(" EHLO——")
	}
}
