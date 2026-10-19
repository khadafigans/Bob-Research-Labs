package notify

import (
	"strings"
	"testing"
)

// hanFreeItems  CJK   (ASCII ··) .
func hanFreeItems(n int) []Item {
	items := make([]Item, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, Item{
			FindingID: int64(i + 1),
			Name:      "login-flaw",
			VulnClass: "SQLi",
			Severity:  "high",
			Summary:   "SQL injection via q param",
			Assets:    []string{"a.example.com"},
		})
	}
	return items
}

// TestMarkdownTitleLocalized  markdown  (dingtalk·wecom) 
//  feishu·html·telegram·webhook    markdownTitle  
func TestMarkdownTitleLocalized(t *testing.T) {
	// (): "Vulnerability digest ·  N"
	got := markdownTitle(Message{Batch: true, Items: hanFreeItems(3)})
	assertKorean(t, "markdownTitle(batch)", got)
	if !strings.Contains(got, "Vulnerability digest") || !strings.Contains(got, "3 total") {
		t.Errorf("  'Vulnerability digest · 3 total'  ,   %q", got)
	}
	//  : "Vulnerability alert"
	empty := markdownTitle(Message{})
	assertKorean(t, "markdownTitle(empty)", empty)
	if empty != "Vulnerability alert" {
		t.Errorf("   'Vulnerability alert'  ,   %q", empty)
	}
}

// TestMarkdownBatchIntroLocalized   (·· )
func TestMarkdownBatchIntroLocalized(t *testing.T) {
	items := hanFreeItems(3)

	//  : " N new finding(s) N"
	win := markdownBatchIntro(Message{Batch: true, WindowMinutes: 30}, items, 3)
	if hasHan(win) {
		t.Errorf("    : %q", win)
	}
	if !strings.Contains(win, "last 30 min") || !strings.Contains(win, "3 new finding(s)") {
		t.Errorf("  '3 new finding(s) in last 30 min'  ,   %q", win)
	}

	//  : "new finding(s) N"(  )
	noWin := markdownBatchIntro(Message{Batch: true}, items, 3)
	if hasHan(noWin) {
		t.Errorf("   : %q", noWin)
	}
	if !strings.Contains(noWin, "3 new finding(s)") || strings.Contains(noWin, "") {
		t.Errorf("   '3 new finding(s)'(  ) ,   %q", noWin)
	}

	//   : "(   N …  N next message …)"
	trunc := markdownBatchIntro(Message{Batch: true, WindowMinutes: 30}, items, 5)
	if hasHan(trunc) {
		t.Errorf("    : %q", trunc)
	}
	for _, want := range []string{"showing first 3", "remaining 2", "follow"} {
		if !strings.Contains(trunc, want) {
			t.Errorf("  %q   ,   %q", want, trunc)
		}
	}
}

// TestWriteItemLabelsLocalized    (markdown 3  )
//  (Status change···)    .
func TestWriteItemLabelsLocalized(t *testing.T) {
	it := Item{
		Name:       "login-flaw",
		VulnClass:  "SQLi",
		Severity:   "high",
		Summary:    "SQL injection via q param",
		Assets:     []string{"a.example.com"},
		DetailURL:  "https://platform.example/finding/1",
		FromStatus: "pending",
		ToStatus:   "fixed",
	}
	var b strings.Builder
	writeItem(&b, it, "", true)
	got := b.String()

	assertKorean(t, "writeItem(single)", got)
	for _, want := range []string{
		"**Status change**: Pending → Fixed",
		"**Type**: SQLi",
		"**Asset**: a.example.com",
		"**Summary**: SQL injection via q param",
		"[View details](https://platform.example/finding/1)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("  %q   :\n%s", want, got)
		}
	}
}

// TestMarkdownBodyFooterLocalized       
func TestMarkdownBodyFooterLocalized(t *testing.T) {
	m := Message{Batch: true, HomeURL: "https://platform.example", Items: hanFreeItems(2)}
	body, kept := markdownBody(m, 0)
	if kept != 2 {
		t.Fatalf(" (0) 2   ,   %d", kept)
	}
	if !strings.Contains(body, "[View all on platform](https://platform.example)") {
		t.Errorf("  'View all on platform'   :\n%s", body)
	}
}
