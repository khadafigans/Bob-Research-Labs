package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// telegramTextLimit  Telegram sendMessage  text （）。
const telegramTextLimit = 4096

// telegramChannel  Telegram Bot API。
//
//   -  URL path （/bot<token>/sendMessage），。
//   -  HTML  MarkdownV2：MarkdownV2  `_*[]()~`>#+-=|{}.!`
type telegramChannel struct{}

func (telegramChannel) Kind() string { return KindTelegram }

func (telegramChannel) DefaultRatePerMin() int { return 20 }

// Bot Token ；chat_id ，（ Token ）。
func (telegramChannel) SecretKeys() []string { return []string{"bot_token"} }

// base_url  Token  API （）， Token。
func (telegramChannel) DestinationKeys() []string { return []string{"base_url"} }

func (telegramChannel) Validate(cfg map[string]any) error {
	if cfgString(cfg, "bot_token") == "" {
		return errors.New("Bot Token is required")
	}
	if cfgString(cfg, "chat_id") == "" {
		return errors.New("Chat ID is required")
	}
	if base := cfgString(cfg, "base_url"); base != "" {
		if err := validateHTTPURL(base); err != nil {
			return fmt.Errorf("Invalid API base URL: %w", err)
		}
	}
	return nil
}

func (c telegramChannel) Send(ctx context.Context, cfg map[string]any, m Message) (int, error) {
	if err := c.Validate(cfg); err != nil {
		return 0, Permanent(err)
	}
	endpoint, err := telegramEndpoint(cfg)
	if err != nil {
		return 0, Permanent(err)
	}
	text, kept := telegramHTML(m)
	payload := map[string]any{
		"chat_id":                  cfgString(cfg, "chat_id"),
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": false,
	}
	raw, err := doJSON(ctx, "POST", endpoint, nil, payload)
	if err != nil {
		return 0, err
	}
	var res struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return 0, fmt.Errorf("Failed to parse Telegram response: %w (%s)", err, snippet(raw))
	}
	if res.OK {
		return kept, nil
	}
	if res.ErrorCode == 429 {
		return 0, fmt.Errorf("Telegram rate limited: %s", res.Description)
	}
	return 0, Permanent(fmt.Errorf("Telegram API error %d: %s", res.ErrorCode, res.Description))
}

// telegramEndpoint  sendMessage 。base_url  API，
func telegramEndpoint(cfg map[string]any) (string, error) {
	base := cfgString(cfg, "base_url")
	if base == "" {
		base = "https://api.telegram.org"
	}
	base = strings.TrimSuffix(base, "/")
	token := cfgString(cfg, "bot_token")
	raw := base + "/bot" + token + "/sendMessage"
	u, err := url.Parse(raw)
	if err != nil {
		//  err： Bot Token， addr 。
		return "", fmt.Errorf("Failed to build API URL (base: %s)", redactRequestTarget(base))
	}
	return u.String(), nil
}

// telegramHTML  HTML ，（ Channel.Send）。
func telegramHTML(m Message) (string, int) {
	var b strings.Builder
	b.WriteString("<b>" + telegramEscape(markdownTitle(m)) + "</b>\n")
	if m.Batch {
		// Telegram ****，（runeSize）。
		footer := ""
		if m.HomeURL != "" {
			footer = fmt.Sprintf("\n\n<a href=\"%s\">View all on platform</a>", telegramEscapeAttr(m.HomeURL))
		}
		kept := packItemCount(m.Items, telegramTextLimit, telegramReservedRunes, footer, runeSize, func(it Item, idx int) string {
			return telegramBatchLine(it, idx+1)
		})
		items := m.Items[:kept]
		b.Reset()
		b.WriteString("<b>" + telegramEscape(telegramBatchTitle(m, items, len(m.Items))) + "</b>")
		for i, it := range items {
			b.WriteString("\n" + telegramEscape(telegramBatchLine(it, i+1)))
		}
		b.WriteString(footer)
		return TruncateHTML(b.String(), telegramTextLimit), kept
	}
	if len(m.Items) == 0 {
		return b.String(), 0
	}
	it := m.Items[0]
	if it.IsStatusChange() {
		b.WriteString(fmt.Sprintf("\n<b>Status change</b>: %s → %s",
			telegramEscape(StatusLabel(it.FromStatus)), telegramEscape(StatusLabel(it.ToStatus))))
	}
	if it.VulnClass != "" && it.VulnClass != it.Title() {
		b.WriteString("\n<b>Type</b>: " + telegramEscape(it.VulnClass))
	}
	if a := assetLine(it.Assets, maxAssetsShown); a != "" {
		b.WriteString("\n<b>Asset</b>: " + telegramEscape(a))
	}
	if s := OneLine(it.Summary, maxSummaryRunes); s != "" {
		b.WriteString("\n<b>Summary</b>: " + telegramEscape(s))
	}
	if it.DetailURL != "" {
		b.WriteString(fmt.Sprintf("\n\n<a href=\"%s\">View details</a>", telegramEscapeAttr(it.DetailURL)))
	}
	return TruncateHTML(b.String(), telegramTextLimit), 1
}

// telegramReservedRunes （）。
const telegramReservedRunes = 160

// telegramBatchLine （，）。
func telegramBatchLine(it Item, idx int) string {
	if a := assetLine(it.Assets, maxAssetsShown); a != "" {
		return fmt.Sprintf("%d. %s · %s — %s", idx, SeverityLabel(it.Severity), it.Title(), a)
	}
	return fmt.Sprintf("%d. %s · %s", idx, SeverityLabel(it.Severity), it.Title())
}

// telegramBatchTitle 。****，
func telegramBatchTitle(m Message, items []Item, total int) string {
	title := fmt.Sprintf("Vulnerability digest · %d total", total)
	if extra := total - len(items); extra > 0 {
		title += fmt.Sprintf(" (showing first %d, remaining %d in next message)", len(items), extra)
	}
	if m.WindowMinutes > 0 {
		title = fmt.Sprintf("Last %d min · %s", m.WindowMinutes, title)
	}
	return title
}

// telegramEscape  HTML 。
func telegramEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// telegramEscapeAttr  HTML 。——
func telegramEscapeAttr(s string) string {
	s = telegramEscape(s)
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
