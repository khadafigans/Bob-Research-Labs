package notify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"
)

// dingTalkChannel 。
//
//     ， DefaultRatePerMin。
//   - / HTTP 200， body  errcode —— errcode
type dingTalkChannel struct{}

func (dingTalkChannel) Kind() string { return KindDingTalk }

func (dingTalkChannel) DefaultRatePerMin() int { return 20 }

//  Webhook  access_token，，。
func (dingTalkChannel) SecretKeys() []string { return []string{"webhook", "secret"} }

func (dingTalkChannel) DestinationKeys() []string { return []string{"webhook"} }

func (dingTalkChannel) Validate(cfg map[string]any) error {
	hook := cfgString(cfg, "webhook")
	if hook == "" {
		return errors.New("Webhook URL is required")
	}
	if err := validateHTTPURL(hook); err != nil {
		return fmt.Errorf("Invalid Webhook URL: %w", err)
	}
	return nil
}

// Send 。 ActionCard（）， markdown。
func (c dingTalkChannel) Send(ctx context.Context, cfg map[string]any, m Message) (int, error) {
	hook := cfgString(cfg, "webhook")
	if err := c.Validate(cfg); err != nil {
		return 0, Permanent(err)
	}
	endpoint, err := dingTalkSignedURL(hook, cfgString(cfg, "secret"), time.Now())
	if err != nil {
		return 0, Permanent(err)
	}

	title := markdownTitle(m)
	text, kept := markdownBody(m, 20000)

	var payload any
	if !m.Batch && len(m.Items) == 1 && m.Items[0].DetailURL != "" {
		payload = map[string]any{
			"msgtype": "actionCard",
			"actionCard": map[string]any{
				"title":          title,
				"text":           text,
				"btnOrientation": "0",
				"singleTitle":    "View details",
				"singleURL":      m.Items[0].DetailURL,
			},
		}
	} else {
		payload = map[string]any{
			"msgtype":  "markdown",
			"markdown": map[string]any{"title": title, "text": text},
		}
	}

	raw, err := doJSON(ctx, "POST", endpoint, nil, payload)
	if err != nil {
		return 0, err
	}
	var res struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return 0, fmt.Errorf("Failed to parse DingTalk response: %w (%s)", err, snippet(raw))
	}
	if res.ErrCode != 0 {
		return 0, Permanent(fmt.Errorf("DingTalk error (%d): %s", res.ErrCode, res.ErrMsg))
	}
	return kept, nil
}

// dingTalkSignedURL  webhook  timestamp  sign 。
//
// ： = timestamp + "\n" + secret，HMAC-SHA256 ** secret**，
//  base64  URL 。timestamp 。secret ，
func dingTalkSignedURL(hook, secret string, now time.Time) (string, error) {
	if secret == "" {
		return hook, nil
	}
	ts := strconv.FormatInt(now.UnixMilli(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "\n" + secret))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	u, err := url.Parse(hook)
	if err != nil {
		//  err：url.Parse （ access_token）。
		return "", fmt.Errorf("Failed to parse Webhook URL: %s", redactRequestTarget(hook))
	}
	q := u.Query()
	q.Set("timestamp", ts)
	q.Set("sign", sign)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// validateHTTPURL 、， IP 。
//
//
//  1. ****。url.Parse  *url.Error， Error() 
//      key、Telegram  bot token、 hook id）。 `return err`，
//
//  2. ** IP **，（blockInternalDial 
//
// ：file:///gopher://  http.Client 
func validateHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("Cannot resolve address (%s)", redactRequestTarget(raw))
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("Only http or https is supported, got: %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("Hostname is required")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && isBlockedDialIP(ip) && !allowLocalTargets() {
		return fmt.Errorf("Refusing to send to local/link-local address %s (set %s=1 to allow local targets)", ip, AllowLocalTargetsEnv)
	}
	return nil
}
