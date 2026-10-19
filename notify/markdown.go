package notify

import (
	"fmt"
	"strings"
)

//  JSON、Telegram  HTML、 HTML，。

// maxAssetsShown 。，
const maxAssetsShown = 3

// maxSummaryRunes 。IM 「」，
const maxSummaryRunes = 120

// markdownReservedBytes （ +  + ）
const markdownReservedBytes = 320

// markdownEscape  markdown 。
//
//
//	 SQL \n[：](http://attacker.tld)
//
// `![](http://attacker.tld/beacon)` ，
//
func markdownEscape(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		"`", "\\`",
		"*", `\*`,
		"_", `\_`,
		"[", `\[`,
		"]", `\]`,
		"(", `\(`,
		")", `\)`,
		"!", `\!`,
		"#", `\#`,
		">", `\>`,
		"|", `\|`,
		"~", `\~`,
	)
	return replacer.Replace(s)
}

// markdownText ， markdown 。
func markdownText(s string, maxRunes int) string {
	return markdownEscape(OneLine(s, maxRunes))
}

// markdownTitle （IM /），****。
//
// ：——markdown 、Telegram 
// HTML、 plain_text、 Webhook  JSON 。
// （markdown  HTML ， JSON 
// ），， writeItem / feishuItemLines /
// telegramEscape。 markdown ， Telegram 
func markdownTitle(m Message) string {
	if m.Batch {
		return fmt.Sprintf("Vulnerability digest · %d total", len(m.Items))
	}
	if len(m.Items) == 0 {
		return "Vulnerability alert"
	}
	it := m.Items[0]
	return fmt.Sprintf("[%s] %s", SeverityLabel(it.Severity), OneLine(it.Title(), 0))
}

// markdownBody ，****。
//
//
func markdownBody(m Message, maxBytes int) (string, int) {
	if !m.Batch {
		if len(m.Items) == 0 {
			return "", 0
		}
		var b strings.Builder
		writeItem(&b, m.Items[0], "", true)
		return TruncateBytes(b.String(), maxBytes), 1
	}

	footer := ""
	if m.HomeURL != "" {
		footer = fmt.Sprintf("\n[View all on platform](%s)\n", m.HomeURL)
	}
	kept := packItemCount(m.Items, maxBytes, markdownReservedBytes, footer, byteSize, func(it Item, idx int) string {
		var b strings.Builder
		writeItem(&b, it, fmt.Sprintf("%d. ", idx+1), false)
		return b.String()
	})

	items := m.Items[:kept]
	var b strings.Builder
	b.WriteString(markdownBatchIntro(m, items, len(m.Items)))
	for i, it := range items {
		writeItem(&b, it, fmt.Sprintf("%d. ", i+1), false)
	}
	b.WriteString(footer)
	return TruncateBytes(b.String(), maxBytes), kept
}

// markdownBatchIntro ：、。
//
func markdownBatchIntro(m Message, items []Item, total int) string {
	var b strings.Builder
	if m.WindowMinutes > 0 {
		fmt.Fprintf(&b, "**%d new finding(s) in last %d min**", total, m.WindowMinutes)
	} else {
		fmt.Fprintf(&b, "**%d new finding(s)**", total)
	}
	if extra := total - len(items); extra > 0 {
		fmt.Fprintf(&b, " (showing first %d in this message; remaining %d follow)", len(items), extra)
	}
	counts := map[string]int{}
	for _, it := range items {
		counts[it.Severity]++
	}
	var parts []string
	for _, sev := range []string{"critical", "high", "medium", "low"} {
		if n := counts[sev]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", SeverityLabel(sev), n))
		}
	}
	if len(parts) > 0 {
		b.WriteString("\n" + strings.Join(parts, " · "))
	}
	b.WriteString("\n\n")
	return b.String()
}

//
// prefix ；single=true （），
//
// （///） markdownText：
//  + 。 public_base_url ，，
func writeItem(b *strings.Builder, it Item, prefix string, single bool) {
	line := fmt.Sprintf("%s**%s · %s**", prefix, SeverityLabel(it.Severity), markdownText(it.Title(), 0))
	if !single {
		var extras []string
		if a := assetLine(it.Assets, maxAssetsShown); a != "" {
			extras = append(extras, markdownText(a, 0))
		}
		if it.Summary != "" {
			extras = append(extras, markdownText(it.Summary, 60))
		}
		if len(extras) > 0 {
			line += " — " + strings.Join(extras, " · ")
		}
		b.WriteString(line + "\n")
		return
	}
	b.WriteString(line + "\n")
	if it.IsStatusChange() {
		fmt.Fprintf(b, "**Status change**: %s → %s\n",
			markdownText(StatusLabel(it.FromStatus), 0), markdownText(StatusLabel(it.ToStatus), 0))
	}
	if it.VulnClass != "" && it.VulnClass != it.Title() {
		fmt.Fprintf(b, "**Type**: %s\n", markdownText(it.VulnClass, 0))
	}
	if a := assetLine(it.Assets, maxAssetsShown); a != "" {
		fmt.Fprintf(b, "**Asset**: %s\n", markdownText(a, 0))
	}
	if it.Summary != "" {
		if s := markdownText(it.Summary, maxSummaryRunes); s != "" {
			fmt.Fprintf(b, "**Summary**: %s\n", s)
		}
	}
	if it.DetailURL != "" {
		fmt.Fprintf(b, "[View details](%s)\n", it.DetailURL)
	}
}
