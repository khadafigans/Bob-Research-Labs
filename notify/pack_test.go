package notify

import (
	"strings"
	"testing"
	"unicode/utf8"
)

//

func TestMarkdownBodyPacksWholeItemsWithinByteLimit(t *testing.T) {
	m := batchMsg(200)
	body, kept := markdownBody(m, weComMarkdownLimit)

	if len(body) > weComMarkdownLimit {
		t.Fatalf(" %d  %d", len(body), weComMarkdownLimit)
	}
	if !utf8.ValidString(body) {
		t.Fatal(" UTF-8")
	}
	if kept <= 0 || kept >= len(m.Items) {
		t.Fatalf("（0 < kept < %d）， %d", len(m.Items), kept)
	}
	if !strings.Contains(body, "remaining") || !strings.Contains(body, "follow") {
		t.Fatalf(":\n%s", body[:minInt(400, len(body))])
	}
	for i := 0; i < kept; i++ {
		if !strings.Contains(body, ""+itoa(i+1)) {
			t.Fatalf(" %d :\n%s", i+1, body)
		}
	}
	if strings.Contains(body, ""+itoa(kept+1)) {
		t.Fatalf(" %d （）", kept+1)
	}
}

func TestMarkdownBodyKeepsEverythingWhenUnderLimit(t *testing.T) {
	m := batchMsg(3)
	body, kept := markdownBody(m, 0) // 0 = 
	if kept != len(m.Items) {
		t.Fatalf("， kept=%d", kept)
	}
	if strings.Contains(body, "") {
		t.Fatalf(":\n%s", body)
	}
}

func TestMarkdownBodyAlwaysKeepsAtLeastOneItem(t *testing.T) {
	m := batchMsg(5)
	_, kept := markdownBody(m, 50)
	if kept != 1 {
		t.Fatalf(" 1 ， %d", kept)
	}
}

func TestMarkdownBodySingleReturnsOne(t *testing.T) {
	_, kept := markdownBody(singleMsg(), 4096)
	if kept != 1 {
		t.Fatalf(" 1 ， %d", kept)
	}
	if _, k := markdownBody(Message{}, 4096); k != 0 {
		t.Fatalf(" 0 ， %d", k)
	}
}

func TestTelegramPackingUsesRuneBudget(t *testing.T) {
	m := batchMsg(200)
	text, kept := telegramHTML(m)
	if n := utf8.RuneCountInString(text); n > telegramTextLimit {
		t.Fatalf(" %d  %d", n, telegramTextLimit)
	}
	if kept <= 0 || kept >= len(m.Items) {
		t.Fatalf("， %d", kept)
	}
	if !strings.Contains(text, "next message") {
		t.Fatalf(":\n%.300s", text)
	}
}

func TestFeishuPackingReportsKept(t *testing.T) {
	m := batchMsg(2000)
	_, kept := feishuCard(m)
	if kept <= 0 || kept >= len(m.Items) {
		t.Fatalf("， %d", kept)
	}
}

func TestWebhookAndEmailReportAllItems(t *testing.T) {
	m := batchMsg(7)
	if n := len(m.Items); n != 7 {
		t.Fatal("")
	}
	// ：markdownBody(0) 。
	if _, k := markdownBody(m, 0); k != len(m.Items) {
		t.Fatalf("， %d", k)
	}
}

// TestMarkdownEscapesUntrustedContent 「」。
func TestMarkdownEscapesUntrustedContent(t *testing.T) {
	cases := []struct {
		name  string
		item  Item
		must  []string // （）
		wrong []string // （）
	}{
		{
			name: " + ",
			item: Item{
				Severity: "high",
				Name:     " SQL \n[：](http://attacker.tld)",
			},
			must:  []string{`\[：\]`, `\(http://attacker.tld\)`},
			wrong: []string{"\n[", "\n\n["},
		},
		{
			name: "",
			item: Item{
				Severity: "high",
				Name:     " ![](http://attacker.tld/beacon)",
			},
			must:  []string{`\!`, `\(http://attacker.tld/beacon\)`},
			wrong: []string{"![]("},
		},
		{
			name: "",
			item: Item{
				Severity: "high",
				Name:     "",
				Assets:   []string{"a.com/**>"},
			},
			must:  []string{`\*\*`, `\>`},
			wrong: []string{"**"},
		},
		{
			name: "",
			item: Item{
				Severity: "high",
				Name:     "",
				Summary:  "`code` | ",
			},
			must:  []string{"\\`code\\`", `\|`},
			wrong: []string{"`code`"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Message{Items: []Item{tc.item}}
			// Item  markdown 。
			var b strings.Builder
			writeItem(&b, tc.item, "", true)
			got := b.String()
			for _, want := range tc.must {
				if !strings.Contains(got, want) {
					t.Errorf(" %q:\n%s", want, got)
				}
			}
			for _, bad := range tc.wrong {
				if strings.Contains(got, bad) {
					t.Errorf(" %q（）:\n%s", bad, got)
				}
			}
			_ = m
		})
	}
}

// TestMarkdownEscapeBackslashFirst ：，
func TestMarkdownEscapeBackslashFirst(t *testing.T) {
	if got := markdownEscape(`a\b*c`); got != `a\\b\*c` {
		t.Fatalf("， %q", got)
	}
}

// TestTelegramTitleHasNoMarkdownEscapes ：
// markdown  Telegram  HTML （
func TestTelegramTitleHasNoMarkdownEscapes(t *testing.T) {
	m := Message{Items: []Item{{Severity: "high", Name: "alert(1) **"}}}
	text, _ := telegramHTML(m)
	if strings.Contains(text, `\(`) || strings.Contains(text, `\*`) {
		t.Fatalf("Telegram  markdown :\n%s", text)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
