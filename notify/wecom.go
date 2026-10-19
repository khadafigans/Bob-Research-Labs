package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// weComMarkdownLimit  markdown content （，）。
// ， TruncateBytes 。
const weComMarkdownLimit = 4096

// weComChannel 。
//
//   -  URL  key ，—— webhook 。
//   - markdown content  4096 ****，（）。 3 /，
type weComChannel struct{}

func (weComChannel) Kind() string { return KindWeCom }

func (weComChannel) DefaultRatePerMin() int { return 20 }

//  Webhook （URL  key），——
func (weComChannel) SecretKeys() []string { return []string{"webhook"} }

func (weComChannel) DestinationKeys() []string { return []string{"webhook"} }

func (weComChannel) Validate(cfg map[string]any) error {
	hook := cfgString(cfg, "webhook")
	if hook == "" {
		return errors.New("Webhook URL is required")
	}
	if err := validateHTTPURL(hook); err != nil {
		return fmt.Errorf("Invalid Webhook URL: %w", err)
	}
	return nil
}

func (c weComChannel) Send(ctx context.Context, cfg map[string]any, m Message) (int, error) {
	if err := c.Validate(cfg); err != nil {
		return 0, Permanent(err)
	}
	content, kept := markdownBody(m, weComMarkdownLimit)
	payload := map[string]any{
		"msgtype":  "markdown",
		"markdown": map[string]any{"content": content},
	}
	raw, err := doJSON(ctx, "POST", cfgString(cfg, "webhook"), nil, payload)
	if err != nil {
		return 0, err
	}
	var res struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return 0, fmt.Errorf("Failed to parse WeCom response: %w (%s)", err, snippet(raw))
	}
	if res.ErrCode != 0 {
		if res.ErrCode == 45009 {
			return 0, fmt.Errorf("WeCom rate limited (%d): %s", res.ErrCode, res.ErrMsg)
		}
		return 0, Permanent(fmt.Errorf("WeCom error (%d): %s", res.ErrCode, res.ErrMsg))
	}
	return kept, nil
}
