package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"
)

// allowLocalTargets  / 。
//
// ——doJSON  4xx/5xx  200  last_error，
//
//  ARTEX_NOTIFY_ALLOW_LOCAL=1 。
//
//  AllowLocalTargetsEnv —— server 
const AllowLocalTargetsEnv = "ARTEX_NOTIFY_ALLOW_LOCAL"

func allowLocalTargets() bool {
	v := strings.TrimSpace(os.Getenv(AllowLocalTargetsEnv))
	return v == "1" || strings.EqualFold(v, "true")
}

// isBlockedDialIP  IP 「」。
//
// **** RFC1918 ： Mattermost / SMTP ，
func isBlockedDialIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	// IPv4-mapped IPv6（::ffff:127.0.0.1） IPv4 ，。
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

// blockInternalDial  http.Transport  Control ，****
//
func blockInternalDial(_, address string, _ syscall.RawConn) error {
	if allowLocalTargets() {
		return nil
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("Cannot resolve target host: %q", host)
	}
	if isBlockedDialIP(ip) {
		return fmt.Errorf("Refusing to send to local/link-local address %s (set %s=1 to allow local targets)", ip, AllowLocalTargetsEnv)
	}
	return nil
}

// notifyTransport  Transport 。
//  Clone （、HTTP/2、、proxy ），
var notifyTransport = func() *http.Transport {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Transport{}
	}
	clone := t.Clone()
	clone.DialContext = (&net.Dialer{Timeout: 10 * time.Second, Control: blockInternalDial}).DialContext
	return clone
}()

//
// ****（server  GlobalProxy）：
//
// ；（ access_token、 key、Telegram 
var httpClient = &http.Client{
	Timeout:   15 * time.Second,
	Transport: notifyTransport,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("Too many redirects")
		}
		if len(via) > 0 && req.URL.Host != via[0].URL.Host {
			return fmt.Errorf("Cross-host redirects are not allowed (%s → %s)", via[0].URL.Host, req.URL.Host)
		}
		return nil
	},
}

// respBodyLimit 。，
const respBodyLimit = 8 << 10

//
// payload  nil  body（ GET  body ）。
// headers ， Webhook 。
//
func doJSON(ctx context.Context, method, url string, headers map[string]string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, Permanent(fmt.Errorf("Failed to build request body: %w", err))
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, Permanent(fmt.Errorf("Invalid request URL: %s", redactRequestTarget(url)))
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		//
		// 。：http.Client.Do  *url.Error，
		//  Error()  `Op "URL": `，** URL **
		// （ access_token、 key、 hook id、Telegram /bot<token>/）。
		// ，：notification_deliveries
		return nil, fmt.Errorf("Request failed: %s", redactTransportError(err))
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, respBodyLimit))
	if readErr != nil {
		return nil, fmt.Errorf("Failed to read response: %w", readErr)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return raw, nil
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusRequestTimeout {
		return nil, fmt.Errorf("Remote server rate-limited or timed out (HTTP %d): %s", resp.StatusCode, snippet(raw))
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("Remote server error (HTTP %d): %s", resp.StatusCode, snippet(raw))
	}
	return nil, Permanent(fmt.Errorf("Remote server rejected the request (HTTP %d): %s", resp.StatusCode, snippet(raw)))
}

func snippet(raw []byte) string {
	return OneLine(string(raw), 200)
}

// redactRequestTarget 「scheme://host/…」，。
//
//
//	    query      /robot/send?access_token=xxx
//	    query      /cgi-bin/webhook/send?key=xxx
//	   **** /open-apis/bot/v2/hook/<hook_id>
//	Telegram **** /bot<token>/sendMessage
//
//
func redactRequestTarget(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "(unresolvable address)"
	}
	return u.Scheme + "://" + u.Host + "/…"
}

// redactTransportError ，。
//
// *url.Error  {Op, URL, Err}，Error()  URL 。
func redactTransportError(err error) string {
	var uerr *url.Error
	if errors.As(err, &uerr) {
		host := ""
		if u, parseErr := url.Parse(uerr.URL); parseErr == nil {
			host = u.Host
		}
		if uerr.Err != nil {
			return fmt.Sprintf("%s %s: %s", uerr.Op, host, uerr.Err)
		}
		return fmt.Sprintf("%s %s: unknown error", uerr.Op, host)
	}
	return redactURLsInText(err.Error())
}

// redactURLsInText  http(s) 。
//
func redactURLsInText(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		rest := s[i:]
		if strings.HasPrefix(rest, "http://") || strings.HasPrefix(rest, "https://") {
			end := len(rest)
			if j := strings.IndexAny(rest, " \t\n\"'"); j >= 0 {
				end = j
			}
			b.WriteString(redactRequestTarget(rest[:end]))
			i += end
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
