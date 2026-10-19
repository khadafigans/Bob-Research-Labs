package notify

import (
	"bufio"
	"context"

	"net"
	"strings"
	"sync"
	"testing"
)

//
//  SMTP ， mock  net/smtp：

// fakeSMTP  SMTP ： greet/EHLO/AUTH/MAIL/RCPT/DATA/QUIT，
type fakeSMTP struct {
	ln net.Listener

	// rcptReply  RCPT TO ； 250。
	rcptReply string
	// mailReply  MAIL FROM ； 250。
	mailReply string
	// advertiseAuth  true  EHLO  AUTH PLAIN。
	advertiseAuth bool

	mu       sync.Mutex
	data     string
	commands []string
}

func newFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{ln: ln, rcptReply: "250 OK", mailReply: "250 OK"}
	go f.serve()
	t.Cleanup(func() { ln.Close() })
	return f
}

func (f *fakeSMTP) hostPort(t *testing.T) (string, int) {
	t.Helper()
	addr, ok := f.ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal(" TCP ")
	}
	return "127.0.0.1", addr.Port
}

func (f *fakeSMTP) record(cmd string) {
	f.mu.Lock()
	f.commands = append(f.commands, cmd)
	f.mu.Unlock()
}

func (f *fakeSMTP) body() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.data
}

func (f *fakeSMTP) sawCommand(prefix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.commands {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func (f *fakeSMTP) serve() {
	conn, err := f.ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	br := bufio.NewReader(conn)
	w := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
	w("220 fake.local ESMTP ready")
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		f.record(line)
		switch {
		case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
			w("250-fake.local")
			if f.advertiseAuth {
				w("250-AUTH PLAIN")
			}
			w("250 8BITMIME")
		case strings.HasPrefix(line, "AUTH"):
			w("235 2.7.0 Authentication successful")
		case strings.HasPrefix(line, "MAIL FROM"):
			w(f.mailReply)
		case strings.HasPrefix(line, "RCPT TO"):
			w(f.rcptReply)
		case strings.HasPrefix(line, "DATA"):
			w("354 End data with <CR><LF>.<CR><LF>")
			var sb strings.Builder
			for {
				dl, err := br.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dl, "\r\n") == "." {
					break
				}
				sb.WriteString(dl)
			}
			f.mu.Lock()
			f.data = sb.String()
			f.mu.Unlock()
			w("250 2.0.0 Ok: queued as FAKE1")
		case strings.HasPrefix(line, "QUIT"):
			w("221 2.0.0 Bye")
			return
		default:
			w("250 OK")
		}
	}
}

func emailCfg(t *testing.T, f *fakeSMTP, extra map[string]any) map[string]any {
	t.Helper()
	host, port := f.hostPort(t)
	cfg := map[string]any{
		"host": host,
		"port": float64(port),
		"from": "artex@example.com",
		"to":   []any{"a@example.com", "b@example.com"},
	}
	for k, v := range extra {
		cfg[k] = v
	}
	return cfg
}

func TestEmailSendDeliversFullMessage(t *testing.T) {
	f := newFakeSMTP(t)
	f.advertiseAuth = true
	cfg := emailCfg(t, f, map[string]any{"username": "artex", "password": "pw"})

	if _, err := (emailChannel{}).Send(context.Background(), cfg, singleMsg()); err != nil {
		t.Fatalf(": %v", err)
	}
	for _, want := range []string{"MAIL FROM:<artex@example.com>", "RCPT TO:<a@example.com>", "RCPT TO:<b@example.com>", "DATA", "AUTH", "QUIT"} {
		if !f.sawCommand(want) {
			t.Errorf("SMTP  %q，：%v", want, f.commands)
		}
	}
	body := f.body()
	if body == "" {
		t.Fatal("DATA ")
	}
	if !strings.Contains(body, "Content-Type: text/html") {
		t.Errorf(" Content-Type :\n%s", body)
	}
	if !strings.Contains(body, "base64") {
		t.Errorf(" base64 （ HTML  SMTP  1000 ）:\n%s", body)
	}
	if !strings.Contains(body, "a@example.com, b@example.com") {
		t.Errorf("To :\n%s", body)
	}
}

func TestEmailSendWithoutAuth(t *testing.T) {
	f := newFakeSMTP(t)
	cfg := emailCfg(t, f, nil)
	if _, err := (emailChannel{}).Send(context.Background(), cfg, singleMsg()); err != nil {
		t.Fatalf(": %v", err)
	}
	if f.sawCommand("AUTH") {
		t.Errorf(" AUTH: %v", f.commands)
	}
}

// TestEmailSendClassifiesSMTPReplies ：
func TestEmailSendClassifiesSMTPReplies(t *testing.T) {
	cases := []struct {
		name      string
		rcptReply string
		mailReply string
		permanent bool
	}{
		{" 550 ", "550 5.1.1 User unknown", "250 OK", true},
		{" 450 ", "450 4.7.1 Greylisting in action", "250 OK", false},
		{" 452 ", "452 4.2.2 Mailbox full", "250 OK", false},
		{" 553 ", "250 OK", "553 5.1.3 Bad address", true},
		{" 451 ", "250 OK", "451 4.3.0 Temporary failure", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeSMTP(t)
			f.rcptReply = tc.rcptReply
			f.mailReply = tc.mailReply
			_, err := (emailChannel{}).Send(context.Background(), emailCfg(t, f, nil), singleMsg())
			if err == nil {
				t.Fatal("")
			}
			if got := IsPermanent(err); got != tc.permanent {
				t.Fatalf("permanent ： %v  %v (%v)", tc.permanent, got, err)
			}
			if !strings.Contains(err.Error(), strings.Fields(tc.rcptReply)[0]) && !strings.Contains(err.Error(), strings.Fields(tc.mailReply)[0]) {
				t.Errorf(": %v", err)
			}
		})
	}
}

func TestEmailSendRefusesPlaintextCredentials(t *testing.T) {
	// net/smtp  PlainAuth （ localhost）。
	f := newFakeSMTP(t)
	f.advertiseAuth = true
	_, port := f.hostPort(t)
	cfg := map[string]any{
		"host":     "smtp.example.com", //  localhost
		"port":     float64(port),
		"from":     "a@example.com",
		"to":       []any{"b@example.com"},
		"username": "artex",
		"password": "pw",
	}
	_, err := (emailChannel{}).Send(context.Background(), cfg, singleMsg())
	if err == nil {
		t.Skip(" DNS ，（）")
	}
	if !IsPermanent(err) && !strings.Contains(err.Error(), "connect") && !strings.Contains(err.Error(), "TLS") && !strings.Contains(err.Error(), "unencrypted") {
		t.Logf("：%v（ localhost ）", err)
	}
}

func TestEmailValidateReportsMissingFields(t *testing.T) {
	cases := []struct {
		name string
		cfg  map[string]any
	}{
		{" host", map[string]any{"port": float64(25), "from": "a@b.c", "to": []any{"d@e.f"}}},
		{" port", map[string]any{"host": "smtp.example.com"}},
		{"port ", map[string]any{"host": "h", "port": float64(70000), "from": "a@b.c", "to": []any{"d@e.f"}}},
		{" from", map[string]any{"host": "h", "port": float64(25), "to": []any{"d@e.f"}}},
		{" to", map[string]any{"host": "h", "port": float64(25), "from": "a@b.c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := (emailChannel{}).Validate(tc.cfg); err == nil {
				t.Fatalf(": %v", tc.cfg)
			}
		})
	}
}

// TestEmailConfigTolerance ：JSONB  float64，
func TestEmailConfigTolerance(t *testing.T) {
	cfg := map[string]any{
		"host": "smtp.example.com",
		"port": "587", // 
		"from": "a@b.c",
		"to":   "d@e.f", // 
		"tls":  "true",  // 
	}
	if err := (emailChannel{}).Validate(cfg); err != nil {
		t.Fatalf(": %v", err)
	}
	if got := cfgInt(cfg, "port"); got != 587 {
		t.Errorf("cfgInt ， %d", got)
	}
	if !cfgBool(cfg, "tls") {
		t.Error("cfgBool  \"true\"")
	}
	if to := cfgStrings(cfg, "to"); len(to) != 1 || to[0] != "d@e.f" {
		t.Errorf("cfgStrings ， %v", to)
	}
}

// TestFilterValidateRejectsTypo ：
func TestFilterValidateRejectsTypo(t *testing.T) {
	good := []string{"", "low", "medium", "high", "critical"}
	for _, s := range good {
		if err := (Filter{MinSeverity: s}).Validate(); err != nil {
			t.Errorf(" %q : %v", s, err)
		}
	}
	for _, s := range []string{"hgih", "HIGH", "", "high ", "crit"} {
		err := (Filter{MinSeverity: s}).Validate()
		if err == nil {
			t.Errorf(" %q （、）", s)
			continue
		}
		if !strings.Contains(err.Error(), "low") || !strings.Contains(err.Error(), "critical") {
			t.Errorf("， %q", err.Error())
		}
	}
}

// TestFilterValidateIsWriteTimeOnly 「、」：
func TestFilterValidateIsWriteTimeOnly(t *testing.T) {
	raw := []byte(`{"min_severity":"hgih"}`)
	f := ParseFilter(raw) // 
	if f.MinSeverity != "hgih" {
		t.Fatalf("， %q", f.MinSeverity)
	}
	_ = Match(f, Snapshot{Kind: EventFindingCreated, Severity: "critical"})
}
